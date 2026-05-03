package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		generateCmd := flag.NewFlagSet("generate", flag.ExitOnError)
		pubKeyPath := generateCmd.String("public-key", "", "output path for public key PEM")
		privKeyPath := generateCmd.String("private-key", "", "output path for private key PEM")
		generateCmd.Parse(os.Args[2:])

		if *pubKeyPath == "" || *privKeyPath == "" {
			fmt.Fprintln(os.Stderr, "error: --public-key and --private-key are required")
			os.Exit(1)
		}

		if err := generateKeys(*pubKeyPath, *privKeyPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "sign":
		signCmd := flag.NewFlagSet("sign", flag.ExitOnError)
		binaryPath := signCmd.String("binary", "", "path to binary file to sign")
		signaturePath := signCmd.String("signature", "", "output path for signature")
		privKeyPath := signCmd.String("private-key", "", "path to private key PEM")
		signCmd.Parse(os.Args[2:])

		if *binaryPath == "" || *signaturePath == "" || *privKeyPath == "" {
			fmt.Fprintln(os.Stderr, "error: --binary, --signature, and --private-key are required")
			os.Exit(1)
		}

		if err := signBinary(*binaryPath, *signaturePath, *privKeyPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "verify":
		verifyCmd := flag.NewFlagSet("verify", flag.ExitOnError)
		binaryPath := verifyCmd.String("binary", "", "path to binary file")
		signaturePath := verifyCmd.String("signature", "", "path to signature file")
		pubKeyPath := verifyCmd.String("public-key", "", "path to public key PEM")
		verifyCmd.Parse(os.Args[2:])

		if *binaryPath == "" || *signaturePath == "" || *pubKeyPath == "" {
			fmt.Fprintln(os.Stderr, "error: --binary, --signature, and --public-key are required")
			os.Exit(1)
		}

		if err := verifyBinary(*binaryPath, *signaturePath, *pubKeyPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: sign-tool <command> [options]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  generate  Generate a new Ed25519 key pair")
	fmt.Fprintln(os.Stderr, "  sign      Sign a binary file")
	fmt.Fprintln(os.Stderr, "  verify    Verify a binary file's signature")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Use 'sign-tool <command> --help' for command-specific options.")
}

func generateKeys(pubKeyPath, privKeyPath string) error {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "ED25519 PUBLIC KEY",
		Bytes: privateKey.Public().(ed25519.PublicKey),
	})
	if err := os.WriteFile(pubKeyPath, pubPEM, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "ED25519 PRIVATE KEY",
		Bytes: privateKey.Seed(),
	})
	if err := os.WriteFile(privKeyPath, privPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	hash := sha256.Sum256(privateKey.Public().(ed25519.PublicKey))
	fingerprint := hex.EncodeToString(hash[:])[:16]

	fmt.Printf("key pair generated (fingerprint: %s)\n", fingerprint)
	fmt.Printf("  public key:  %s\n", pubKeyPath)
	fmt.Printf("  private key: %s\n", privKeyPath)
	return nil
}

func signBinary(binaryPath, signaturePath, privKeyPath string) error {
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to read binary: %w", err)
	}

	privBytes, err := os.ReadFile(privKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	block, _ := pem.Decode(privBytes)
	if block == nil {
		return fmt.Errorf("failed to decode private key PEM")
	}

	privateKey := ed25519.NewKeyFromSeed(block.Bytes)
	signature := ed25519.Sign(privateKey, data)

	if err := os.WriteFile(signaturePath, signature, 0644); err != nil {
		return fmt.Errorf("failed to write signature: %w", err)
	}

	fmt.Printf("binary signed: %s\n", binaryPath)
	fmt.Printf("signature:    %s (%d bytes)\n", signaturePath, len(signature))
	return nil
}

func verifyBinary(binaryPath, signaturePath, pubKeyPath string) error {
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to read binary: %w", err)
	}

	sig, err := os.ReadFile(signaturePath)
	if err != nil {
		return fmt.Errorf("failed to read signature: %w", err)
	}

	pubBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	block, _ := pem.Decode(pubBytes)
	if block == nil {
		return fmt.Errorf("failed to decode public key PEM")
	}

	valid := ed25519.Verify(block.Bytes, data, sig)
	if valid {
		fmt.Println("signature: VALID")
	} else {
		fmt.Println("signature: INVALID")
		os.Exit(1)
	}
	return nil
}
