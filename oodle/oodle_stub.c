#include <malloc.h>  // calloc, free, malloc
#include <memory.h>  // memcpy_s
#include <stdbool.h> // bool, false, true
#include <stdio.h>   // sprintf_s
#include <stdlib.h>  // abort

#define WIN32_LEAN_AND_MEAN
#include <Windows.h>
#include <DbgHelp.h>
#include <Psapi.h>

#define HASHTABLE_BITS 19
#define WINDOW_SIZE 0x16000
#define MAX_DECOMPRESSED_SIZE (1 << 16)

static __int64 (*OodleNetworkUDP_State_Size)();

static __int64 (*OodleNetwork1_Shared_Size)(char n);

static __int64 (*OodleNetwork1UDP_Decode)(
    const void *state,
    const void *shared,
    const void *comp,
    __int64 compLen,
    void *raw,
    __int64 rawLen);

static void (*OodleNetwork1_Shared_SetWindow)(
    void *data,
    int htbits,
    const void *windowv,
    int window_size);

static void (*OodleNetwork1UDP_Train)(
    void *state,
    const void *shared,
    const void **training_packet_pointers,
    const int *training_packet_sizes,
    int num_training_packets);

/**
 * @brief Scans an HMODULE's image for a byte pattern.
 * -1 indicates a wildcard (??) byte.
 *
 * @param hModule The module to scan
 * @param pattern The byte pattern to search for
 * @param patternLen The number of elements in pattern
 * @return `void*` The pointer to the beginning of the found pattern, or NULL if not found
 */
static void *scanImage(HMODULE hModule, int pattern[], unsigned long long patternLen)
{
    MODULEINFO modinfo;
    if (!GetModuleInformation(GetCurrentProcess(), hModule, &modinfo, sizeof(modinfo)))
        return NULL;

    // Calculate the search bounds
    unsigned char *start = modinfo.lpBaseOfDll;
    unsigned char *end = start + modinfo.SizeOfImage - patternLen;

    // Scan the image for the pattern
    for (unsigned char *offset = start; offset < end; ++offset)
    {
        bool matched = true;

        for (unsigned long long i = 0; i < patternLen; ++i)
        {
            if ((pattern[i] != -1) && (pattern[i] != offset[i]))
            {
                matched = false;
                break;
            }
        }

        if (matched)
            return offset;
    }

    // No match found in the entire image
    return NULL;
}

// Look at me, I am the PE loader now.
// https://www.codeproject.com/Articles/1045674/Load-EXE-as-DLL-Mission-Possible
static bool fixupImports(HMODULE hModule)
{
    ULONG size;
    PIMAGE_IMPORT_DESCRIPTOR pImportDesc = ImageDirectoryEntryToDataEx(
        hModule,
        TRUE,
        IMAGE_DIRECTORY_ENTRY_IMPORT,
        &size,
        NULL);

    if (!pImportDesc)
        return false;

    for (; pImportDesc->Name; ++pImportDesc)
    {
        PSTR pszModName = (PBYTE)hModule + pImportDesc->Name;
        if (!pszModName)
            break;

        // We only care about kernel32.dll
        // Remove this conditional if we start caring about other DLLs
        if (stricmp(pszModName, "kernel32.dll") != 0)
            continue;

        HINSTANCE hImportDLL = LoadLibraryA(pszModName);
        if (!hImportDLL)
            return false;

        // Get caller's import address table (IAT) for the callee's functions
        PIMAGE_THUNK_DATA pThunk = (PIMAGE_THUNK_DATA)((PBYTE)hModule + pImportDesc->FirstThunk);

        for (; pThunk->u1.Function; ++pThunk)
        {
            FARPROC pfnNew = 0;

            // Get the address of the function address
            PROC *ppfn = (PROC *)&pThunk->u1.Function;
            if (!ppfn)
                return false;

            if (pThunk->u1.Ordinal & IMAGE_ORDINAL_FLAG)
            {
                size_t ord = IMAGE_ORDINAL(pThunk->u1.Ordinal);

                char fe[100] = {0};
                sprintf_s(fe, sizeof(fe), "#%u", ord);

                pfnNew = GetProcAddress(hImportDLL, (LPCSTR)ord);
            }
            else
            {
                PSTR fName = (PSTR)hModule + pThunk->u1.Function + 2;
                if (!fName)
                    break;

                pfnNew = GetProcAddress(hImportDLL, fName);
            }

            if (!pfnNew)
                return false;

            // Make the memory writeable
            DWORD dwOldProtect;
            if (!VirtualProtect(pThunk, sizeof(pfnNew), PAGE_WRITECOPY, &dwOldProtect))
                return false;

            if (memcpy_s(pThunk, sizeof(pfnNew), &pfnNew, sizeof(pfnNew)) != 0)
                return false;

            // Restore the original memory protection
            if (!VirtualProtect(pThunk, sizeof(pfnNew), dwOldProtect, &dwOldProtect))
                return false;
        }
    }

    return true;
}

/**
 * @brief The module handle to the game executable (loaded as a library).
 */
static HMODULE hModule = NULL;

static unsigned char *state = NULL;
static unsigned char *shared = NULL;
static unsigned char *window = NULL;

static unsigned char *scratch = NULL;

DWORD init(const char *lpLibFileName)
{
    // Load the game executable as a library (this is cursed!)
    hModule = LoadLibraryEx(lpLibFileName, NULL, LOAD_LIBRARY_REQUIRE_SIGNED_TARGET);
    if (!hModule)
        return GetLastError();

    // B8 ?? ?? ?? ?? C3 CC CC CC CC CC CC CC CC CC CC 40 55 56
    OodleNetworkUDP_State_Size = scanImage(hModule, (int[]){0xB8, -1, -1, -1, -1, 0xC3, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0x40, 0x55, 0x56}, 19);
    if (!OodleNetworkUDP_State_Size)
        return 1;

    // B8 ?? ?? ?? ?? 48 D3 E0 48 8D 04 C5
    OodleNetwork1_Shared_Size = scanImage(hModule, (int[]){0xB8, -1, -1, -1, -1, 0x48, 0xD3, 0xE0, 0x48, 0x8D, 0x04, 0xC5}, 12);
    if (!OodleNetwork1_Shared_Size)
        return 1;

    // 48 89 5C 24 ?? 48 89 6C 24 ?? 48 89 74 24 ?? 48 89 7C 24 ?? 41 56 48 83 EC 20 41 8B D9 49 8B F0
    OodleNetwork1_Shared_SetWindow = scanImage(hModule, (int[]){0x48, 0x89, 0x5C, 0x24, -1, 0x48, 0x89, 0x6C, 0x24, -1, 0x48, 0x89, 0x74, 0x24, -1, 0x48, 0x89, 0x7C, 0x24, -1, 0x41, 0x56, 0x48, 0x83, 0xEC, 0x20, 0x41, 0x8B, 0xD9, 0x49, 0x8B, 0xF0}, 32);
    if (!OodleNetwork1_Shared_SetWindow)
        return 1;

    // 48 89 5C 24 ?? 48 89 6C 24 ?? 48 89 74 24 ?? 48 89 7C 24 ?? 41 56 48 83 EC 30 48 8B F2
    OodleNetwork1UDP_Train = scanImage(hModule, (int[]){0x48, 0x89, 0x5C, 0x24, -1, 0x48, 0x89, 0x6C, 0x24, -1, 0x48, 0x89, 0x74, 0x24, -1, 0x48, 0x89, 0x7C, 0x24, -1, 0x41, 0x56, 0x48, 0x83, 0xEC, 0x30, 0x48, 0x8B, 0xF2}, 29);
    if (!OodleNetwork1UDP_Train)
        return 1;

    // 40 53 48 83 EC 30 48 8B 44 24 ?? 49 8B D9 48 85 C0
    OodleNetwork1UDP_Decode = scanImage(hModule, (int[]){0x40, 0x53, 0x48, 0x83, 0xEC, 0x30, 0x48, 0x8B, 0x44, 0x24, -1, 0x49, 0x8B, 0xD9, 0x48, 0x85, 0xC0}, 17);
    if (!OodleNetwork1UDP_Decode)
        return 1;

    // Allocate memory for Oodle operations
    // Note: These *must* be calloc, otherwise it will mysteriously crash on decode
    state = calloc(OodleNetworkUDP_State_Size(), 1);
    if (!state)
        return 2;

    shared = calloc(OodleNetwork1_Shared_Size(HASHTABLE_BITS), 1);
    if (!shared)
        return 2;

    window = calloc(WINDOW_SIZE, 1);
    if (!window)
        return 2;

    // Scratch buffer for writing decompressed data to
    scratch = calloc(MAX_DECOMPRESSED_SIZE, 1);
    if (!scratch)
        return 2;

    // Patch the import table (in memory) for the game image,
    // so that it can call imported functions image without crashing.
    if (!fixupImports(hModule))
        return 3;

    // Set up Oodle
    OodleNetwork1_Shared_SetWindow(shared, HASHTABLE_BITS, window, WINDOW_SIZE);
    OodleNetwork1UDP_Train(state, shared, NULL, NULL, 0);

    return 0;
}

void deinit()
{
    free(state);
    state = NULL;

    free(shared);
    shared = NULL;

    free(window);
    window = NULL;

    free(scratch);
    scratch = NULL;

    FreeLibrary(hModule);
    hModule = NULL;
}

void *decode(void *comp, __int64 compLen, __int64 rawLen)
{
    if (rawLen > MAX_DECOMPRESSED_SIZE)
        abort(); // Our assumption is invalid

    if (OodleNetwork1UDP_Decode(state, shared, comp, compLen, scratch, rawLen))
        return scratch; // It is the caller's job to know many bytes to copy (should be `rawLen` bytes)

    return NULL;
}
