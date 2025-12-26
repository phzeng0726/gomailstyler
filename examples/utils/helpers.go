package utils

import (
	"io"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the environment variables for SMTP configuration
type Config struct {
	SMTPServer   string
	SMTPPort     string
	SMTPSender   string
	MailReceiver string
}

// LoadConfig loads environment variables from .env file
// It searches for .env file in current directory and parent directories
func LoadConfig() *Config {
	// Try multiple paths to find .env file
	paths := []string{
		".env",           // current directory
		"../.env",        // parent directory (from example subdirs)
		"../../.env",     // two levels up (from nested subdirs)
		"../../../.env",  // three levels up
	}

	var err error
	loaded := false
	for _, path := range paths {
		err = godotenv.Load(path)
		if err == nil {
			loaded = true
			log.Printf("Loaded .env from: %s", path)
			break
		}
	}

	if !loaded {
		log.Fatal("Error loading .env file. Please create a .env file in the examples/ directory with:\n" +
			"SMTP_SERVER=your-smtp-server\n" +
			"SMTP_PORT=587\n" +
			"SMTP_SENDER=your-email@example.com\n" +
			"MAIL_RECEIVER=recipient@example.com")
	}

	return &Config{
		SMTPServer:   os.Getenv("SMTP_SERVER"),
		SMTPPort:     os.Getenv("SMTP_PORT"),
		SMTPSender:   os.Getenv("SMTP_SENDER"),
		MailReceiver: os.Getenv("MAIL_RECEIVER"),
	}
}

// FileToBytes reads a file and returns its contents as bytes
func FileToBytes(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}
