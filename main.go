package main

import (
	"github.com/sirupsen/logrus"
	"github.com/sparta142/goblade/cmd"
)

func main() {
	logrus.StandardLogger().Formatter = &logrus.TextFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	}

	cmd.Execute()
}
