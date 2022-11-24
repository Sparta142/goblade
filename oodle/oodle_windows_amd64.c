#include <assert.h>   // assert, static_assert
#include <malloc.h>   // _aligned_free, _aligned_malloc
#include <memory.h>   // memcpy_s
#include <stdalign.h> // alignas, alignof
#include <stdbool.h>  // bool, false, true
#include <stddef.h>   // size_t
#include <stdint.h>   // int32_t, int64_t
#include <stdio.h>    // sprintf_s
#include <string.h>   // memset

#define WIN32_LEAN_AND_MEAN
#include <windows.h> // ...
#include <dbghelp.h> // ImageDirectoryEntryToDataEx
#include <psapi.h>   // GetModuleInformation, MODULEINFO

#include <xmmintrin.h> // __m128

#define HASHTABLE_BITS 19
#define WINDOW_SIZE 0x16000
#define MAX_DECOMPRESSED_SIZE (1 << 16)

static int64_t (*OodleNetworkUDP_State_Size)() = NULL;

static int64_t (*OodleNetwork1_Shared_Size)(char n) = NULL;

static int64_t (*OodleNetwork1UDP_Decode)(
    const void *state,
    const void *shared,
    const void *comp,
    int64_t compLen,
    void *raw,
    int64_t rawLen) = NULL;

static void (*OodleNetwork1_Shared_SetWindow)(
    void *data,
    int32_t htbits,
    const void *windowv,
    int32_t window_size) = NULL;

static void (*OodleNetwork1UDP_Train)(
    void *state,
    const void *shared,
    const void **training_packet_pointers,
    const int32_t *training_packet_sizes,
    int32_t num_training_packets) = NULL;

/**
 * @brief Scans an HMODULE's image for a byte pattern.
 * -1 indicates a wildcard (??) byte.
 *
 * @param hModule The module to scan
 * @param pattern The byte pattern to search for
 * @param patternLen The number of elements in pattern
 * @return `void*` The pointer to the beginning of the found pattern, or NULL if not found
 */
static void *scan_image(const HMODULE hModule, const int pattern[], const size_t patternLen)
{
    MODULEINFO modinfo;
    if (!GetModuleInformation(GetCurrentProcess(), hModule, &modinfo, sizeof(modinfo)))
        return NULL;

    // Calculate the search bounds
    const unsigned char *const start = modinfo.lpBaseOfDll;
    const unsigned char *const end = start + modinfo.SizeOfImage - patternLen;

    // Scan the image for the pattern
    for (unsigned char *offset = start; offset < end; ++offset)
    {
        bool matched = true;

        for (size_t i = 0; i < patternLen; ++i)
        {
            if ((pattern[i] != -1) && (pattern[i] != offset[i]))
            {
                // The desired byte is not a wildcard and it does not match the candidate byte
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
static bool fixup_imports(HMODULE hModule)
{
    ULONG size;
    const IMAGE_IMPORT_DESCRIPTOR *pImportDesc = ImageDirectoryEntryToDataEx(
        hModule,
        true,
        IMAGE_DIRECTORY_ENTRY_IMPORT,
        &size,
        NULL);

    if (!pImportDesc)
        return false;

    for (; pImportDesc->Name; ++pImportDesc)
    {
        const PCSTR pszModName = (PCHAR)hModule + pImportDesc->Name;
        if (!pszModName)
            break;

        // We only care about kernel32.dll
        // Remove this conditional if we start caring about other DLLs
        if (stricmp(pszModName, "kernel32.dll") != 0)
            continue;

        const HINSTANCE hImportDLL = LoadLibraryA(pszModName);
        if (!hImportDLL)
            return false;

        // Get caller's import address table (IAT) for the callee's functions
        for (PIMAGE_THUNK_DATA pThunk = (PIMAGE_THUNK_DATA)((PBYTE)hModule + pImportDesc->FirstThunk); pThunk->u1.Function; ++pThunk)
        {
            FARPROC pfnNew = 0;

            // Get the address of the function address
            const PROC *ppfn = (PROC *)&pThunk->u1.Function;
            if (!ppfn)
                return false;

            if (pThunk->u1.Ordinal & IMAGE_ORDINAL_FLAG)
            {
                size_t ord = IMAGE_ORDINAL(pThunk->u1.Ordinal);

                char fe[100] = {0};
                sprintf_s(fe, sizeof(fe), "#%zu", ord);

                pfnNew = GetProcAddress(hImportDLL, fe);
            }
            else
            {
                const PCSTR fName = (PCSTR)hModule + pThunk->u1.Function + 2;
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
 * @brief Allocates a zero-initialized, aligned buffer of the specified size.
 *
 * @param size The number of bytes to allocate.
 * @return `void*` The pointer to the beginning of newly allocated memory.
 */
static void *aligned_calloc(const size_t size)
{
    return memset(_aligned_malloc(size, alignof(__m128)), 0, size);
}

/**
 * @brief The module handle to the game executable (loaded as a library).
 */
static HMODULE hModule = NULL;

/**
 * @brief A critical section object guarding the use of OodleNetwork1UDP_Decode.
 */
static CRITICAL_SECTION criticalSection;

static void *window = NULL;
static void *state = NULL;
static void *shared = NULL;

DWORD setup(const LPCSTR lpLibFileName)
{
    assert(lpLibFileName != NULL);

    if (hModule != NULL)
        return 0;

    InitializeCriticalSectionAndSpinCount(&criticalSection, 0x400);

    // Load the game executable as a library (this is cursed!)
    hModule = LoadLibraryExA(lpLibFileName, NULL, LOAD_LIBRARY_REQUIRE_SIGNED_TARGET);
    if (!hModule)
        return GetLastError();

    // Patch the import table (in memory) for the game image,
    // so that it can call imported functions image without crashing.
    if (!fixup_imports(hModule))
        return 3;

    // B8 ?? ?? ?? ?? C3 CC CC CC CC CC CC CC CC CC CC 40 55 56
    OodleNetworkUDP_State_Size = scan_image(hModule, (int[]){0xB8, -1, -1, -1, -1, 0xC3, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0x40, 0x55, 0x56}, 19);

    // B8 ?? ?? ?? ?? 48 D3 E0 48 8D 04 C5
    OodleNetwork1_Shared_Size = scan_image(hModule, (int[]){0xB8, -1, -1, -1, -1, 0x48, 0xD3, 0xE0, 0x48, 0x8D, 0x04, 0xC5}, 12);

    // 48 89 5C 24 ?? 48 89 6C 24 ?? 48 89 74 24 ?? 48 89 7C 24 ?? 41 56 48 83 EC 20 41 8B D9 49 8B F0
    OodleNetwork1_Shared_SetWindow = scan_image(hModule, (int[]){0x48, 0x89, 0x5C, 0x24, -1, 0x48, 0x89, 0x6C, 0x24, -1, 0x48, 0x89, 0x74, 0x24, -1, 0x48, 0x89, 0x7C, 0x24, -1, 0x41, 0x56, 0x48, 0x83, 0xEC, 0x20, 0x41, 0x8B, 0xD9, 0x49, 0x8B, 0xF0}, 32);

    // 48 89 5C 24 ?? 48 89 6C 24 ?? 48 89 74 24 ?? 48 89 7C 24 ?? 41 56 48 83 EC 30 48 8B F2
    OodleNetwork1UDP_Train = scan_image(hModule, (int[]){0x48, 0x89, 0x5C, 0x24, -1, 0x48, 0x89, 0x6C, 0x24, -1, 0x48, 0x89, 0x74, 0x24, -1, 0x48, 0x89, 0x7C, 0x24, -1, 0x41, 0x56, 0x48, 0x83, 0xEC, 0x30, 0x48, 0x8B, 0xF2}, 29);

    // 40 53 48 83 EC 30 48 8B 44 24 ?? 49 8B D9 48 85 C0
    OodleNetwork1UDP_Decode = scan_image(hModule, (int[]){0x40, 0x53, 0x48, 0x83, 0xEC, 0x30, 0x48, 0x8B, 0x44, 0x24, -1, 0x49, 0x8B, 0xD9, 0x48, 0x85, 0xC0}, 17);

    if (!OodleNetworkUDP_State_Size || !OodleNetwork1_Shared_Size || !OodleNetwork1_Shared_SetWindow || !OodleNetwork1UDP_Train || !OodleNetwork1UDP_Decode)
        return 1;

    // Allocate memory for Oodle operations.
    // These *must* be zero-initialized and aligned,
    // otherwise it will mysteriously crash.
    window = aligned_calloc(WINDOW_SIZE);
    state = aligned_calloc(OodleNetworkUDP_State_Size());
    shared = aligned_calloc(OodleNetwork1_Shared_Size(HASHTABLE_BITS));

    if (!window || !state || !shared)
        return 2;

    // Set up Oodle
    EnterCriticalSection(&criticalSection);
    OodleNetwork1_Shared_SetWindow(shared, HASHTABLE_BITS, window, WINDOW_SIZE);
    OodleNetwork1UDP_Train(state, shared, NULL, NULL, 0);
    LeaveCriticalSection(&criticalSection);

    return 0;
}

void shutdown()
{
    _aligned_free(shared);
    shared = NULL;

    _aligned_free(state);
    state = NULL;

    _aligned_free(window);
    window = NULL;

    OodleNetwork1UDP_Decode = NULL;
    OodleNetwork1UDP_Train = NULL;
    OodleNetwork1_Shared_SetWindow = NULL;
    OodleNetwork1_Shared_Size = NULL;
    OodleNetworkUDP_State_Size = NULL;

    FreeLibrary(hModule);
    hModule = NULL;

    DeleteCriticalSection(&criticalSection);
}

bool decode(const void *comp, const int64_t compLen, void *raw, const int64_t rawLen)
{
    assert(rawLen <= MAX_DECOMPRESSED_SIZE);

    // Copy the compressed data into aligned storage
    alignas(__m128) unsigned char compAligned[MAX_DECOMPRESSED_SIZE];
    if (memcpy_s(compAligned, MAX_DECOMPRESSED_SIZE, comp, compLen) != 0)
        return false;

    // Decompress the data
    EnterCriticalSection(&criticalSection);
    const bool success = OodleNetwork1UDP_Decode(state, shared, compAligned, compLen, raw, rawLen);
    LeaveCriticalSection(&criticalSection);

    return success;
}

static_assert(
    sizeof(void *) == sizeof(OodleNetworkUDP_State_Size),
    "void pointer and function pointer are different sizes");
