package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/license"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "issue":
		issueCmd := flag.NewFlagSet("issue", flag.ExitOnError)
		tenant := issueCmd.String("tenant", "", "tenant ID (required)")
		tier := issueCmd.String("tier", "community", "license tier: community, team, pro, enterprise")
		useType := issueCmd.String("use-type", "non_commercial", "usage type: non_commercial, commercial")
		expires := issueCmd.String("expires", "", "expiration date (RFC3339, e.g. 2026-12-31T23:59:59Z)")
		maxServers := issueCmd.Int("max-servers", 0, "max servers (0 = tier default)")
		maxApps := issueCmd.Int("max-apps", 0, "max apps (0 = tier default)")
		maxUsers := issueCmd.Int("max-users", 0, "max users (0 = tier default)")
		privateKeyFile := issueCmd.String("private-key", "", "path to Ed25519 private key file (required)")
		addons := issueCmd.String("addons", "", "comma-separated addon keys (e.g. feature:dashboard_tv,resource:servers:10)")
		issuerRole := issueCmd.String("issuer-role", "developer", "issuer role: developer, distributor")
		issuedTo := issueCmd.String("issued-to", "", "distributor tenant ID (when developer issues)")
		maxIssued := issueCmd.Int("max-issued", 0, "max sub-licenses a distributor can issue")
		_, err := issueCmd.Parse(os.Args[2:]); if err != nil { fmt.Fprintf(os.Stderr, "error: %v\n", err); os.Exit(1) }

		if *tenant == "" {
			fmt.Fprintln(os.Stderr, "error: --tenant is required")
			os.Exit(1)
		}
		if *privateKeyFile == "" {
			fmt.Fprintln(os.Stderr, "error: --private-key is required")
			os.Exit(1)
		}

		privKey, err := loadPrivateKey(*privateKeyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// Get tier limits
		limits, ok := license.TierLimits[license.Tier(*tier)]
		if !ok {
			fmt.Fprintf(os.Stderr, "error: unknown tier %q\n", *tier)
			os.Exit(1)
		}

		maxSrv := limits.MaxServers
		maxApp := limits.MaxApps
		maxUsr := limits.MaxUsers
		if *maxServers > 0 {
			maxSrv = *maxServers
		}
		if *maxApps > 0 {
			maxApp = *maxApps
		}
		if *maxUsers > 0 {
			maxUsr = *maxUsers
		}

		// Parse expiration
		var expiresAt int64
		if *expires != "" {
			t, err := time.Parse(time.RFC3339, *expires)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: invalid expires format (use RFC3339): %v\n", err)
				os.Exit(1)
			}
			expiresAt = t.Unix()
		}

		// Parse addons
		var addonList []license.Addon
		if *addons != "" {
			for _, key := range strings.Split(*addons, ",") {
				key = strings.TrimSpace(key)
				if key != "" {
					addonList = append(addonList, license.Addon{
						Key:         key,
						Amount:      0,
						PurchasedAt: time.Now().Unix(),
					})
				}
			}
		}

		data := license.LicenseData{
			TenantID:   *tenant,
			UseType:    *useType,
			Tier:       *tier,
			IssuerRole: *issuerRole,
			IssuedTo:   *issuedTo,
			MaxIssued:  *maxIssued,
			Addons:     addonList,
			MaxServers: maxSrv,
			MaxApps:    maxApp,
			MaxUsers:   maxUsr,
			IssuedAt:   time.Now().Unix(),
			ExpiresAt:  expiresAt,
		}

		licenseKey, err := license.GenerateLicenseKey(privKey, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to generate license key: %v\n", err)
			os.Exit(1)
		}

		// Output
		output := map[string]interface{}{
			"license_key": licenseKey,
			"tenant_id":   *tenant,
			"tier":        *tier,
			"use_type":    *useType,
			"max_servers": maxSrv,
			"max_apps":    maxApp,
			"max_users":   maxUsr,
			"issued_at":   time.Now().Format(time.RFC3339),
		}
		if expiresAt > 0 {
			output["expires_at"] = time.Unix(expiresAt, 0).Format(time.RFC3339)
		}
		if len(addonList) > 0 {
			output["addons"] = addonList
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(output); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to encode output: %v\n", err)
			os.Exit(1)
		}

	case "verify":
		verifyCmd := flag.NewFlagSet("verify", flag.ExitOnError)
		key := verifyCmd.String("key", "", "license key to verify (required)")
		publicKeyFile := verifyCmd.String("public-key", "", "path to Ed25519 public key file (required)")
		_, err := verifyCmd.Parse(os.Args[2:]); if err != nil { fmt.Fprintf(os.Stderr, "error: %v\n", err); os.Exit(1) }

		if *key == "" {
			fmt.Fprintln(os.Stderr, "error: --key is required")
			os.Exit(1)
		}
		if *publicKeyFile == "" {
			fmt.Fprintln(os.Stderr, "error: --public-key is required")
			os.Exit(1)
		}

		pubKey, err := loadPublicKey(*publicKeyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// Create engine and load license
		engine := license.NewEngine(pubKey, 7)
		if err := engine.LoadLicense(*key); err != nil {
			fmt.Fprintf(os.Stderr, "error: license verification failed: %v\n", err)
			os.Exit(1)
		}

		if err := engine.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: license validation failed: %v\n", err)
		}

		info := engine.GetInfo()
		if info == nil {
			fmt.Fprintln(os.Stderr, "error: license info not available")
			os.Exit(1)
		}

		maxSrv, maxApp, maxUsr := engine.GetLimits()

		output := map[string]interface{}{
			"valid":      true,
			"tenant_id":  info.Data.TenantID,
			"tier":       string(info.Tier),
			"use_type":   string(info.UseType),
			"issuer_role": info.Data.IssuerRole,
			"max_servers": maxSrv,
			"max_apps":    maxApp,
			"max_users":   maxUsr,
			"valid_from":  info.ValidFrom.Format(time.RFC3339),
			"addons":      info.Addons,
		}
		if !info.ValidTo.IsZero() {
			output["valid_to"] = info.ValidTo.Format(time.RFC3339)
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(output); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to encode output: %v\n", err)
			os.Exit(1)
		}

	case "generate-keys":
		genCmd := flag.NewFlagSet("generate-keys", flag.ExitOnError)
		outputDir := genCmd.String("output-dir", ".", "output directory for key files")
		_, err := genCmd.Parse(os.Args[2:]); if err != nil { fmt.Fprintf(os.Stderr, "error: %v\n", err); os.Exit(1) }

		pubKey, privKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to generate key pair: %v\n", err)
			os.Exit(1)
		}

		if err := os.MkdirAll(*outputDir, 0700); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to create output directory: %v\n", err)
			os.Exit(1)
		}

		pubPath := filepath.Join(*outputDir, "license_public.pem")
		privPath := filepath.Join(*outputDir, "license_private.pem")

		// Write public key
		pubPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "ED25519 PUBLIC KEY",
			Bytes: pubKey,
		})
		if err := os.WriteFile(pubPath, pubPEM, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to write public key: %v\n", err)
			os.Exit(1)
		}

		// Write private key (seed only)
		privPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "ED25519 PRIVATE KEY",
			Bytes: privKey.Seed(),
		})
		if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to write private key: %v\n", err)
			os.Exit(1)
		}

		// Also write base64 versions for config
		pubB64Path := filepath.Join(*outputDir, "license_public_base64.txt")
		privB64Path := filepath.Join(*outputDir, "license_private_base64.txt")

		if err := os.WriteFile(pubB64Path, []byte(base64.StdEncoding.EncodeToString(pubKey)), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to write public key base64: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(privB64Path, []byte(base64.StdEncoding.EncodeToString(privKey.Seed())), 0600); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to write private key base64: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Ed25519 key pair generated:\n")
		fmt.Printf("  Public key:  %s\n", pubPath)
		fmt.Printf("  Private key: %s\n", privPath)
		fmt.Printf("  Public (b64): %s\n", pubB64Path)
		fmt.Printf("  Private (b64): %s\n", privB64Path)
		fmt.Printf("\nAdd to config.yaml:\n")
		fmt.Printf("  license:\n")
		fmt.Printf("    public_key_base64: %s\n", base64.StdEncoding.EncodeToString(pubKey))
		fmt.Printf("\nKeep the private key secure! It is needed for issuing licenses.\n")

	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "DeployPilot License Keygen Tool\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  license issue         Issue a new license key\n")
	fmt.Fprintf(os.Stderr, "  license verify         Verify a license key\n")
	fmt.Fprintf(os.Stderr, "  license generate-keys  Generate a new Ed25519 key pair\n\n")
	fmt.Fprintf(os.Stderr, "Use 'license <command> --help' for command-specific flags.\n")
}

func loadPrivateKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	// Try PEM format first
	block, _ := pem.Decode(data)
	if block != nil {
		if len(block.Bytes) != ed25519.SeedSize {
			return nil, fmt.Errorf("invalid private key seed size: got %d, want %d", len(block.Bytes), ed25519.SeedSize)
		}
		return ed25519.NewKeyFromSeed(block.Bytes), nil
	}

	// Try raw base64
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key base64: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("invalid private key seed size: got %d, want %d", len(seed), ed25519.SeedSize)
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

func loadPublicKey(path string) (ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	// Try PEM format first
	block, _ := pem.Decode(data)
	if block != nil {
		if len(block.Bytes) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("invalid public key size: got %d, want %d", len(block.Bytes), ed25519.PublicKeySize)
		}
		return ed25519.PublicKey(block.Bytes), nil
	}

	// Try raw base64
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key base64: %w", err)
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: got %d, want %d", len(keyBytes), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(keyBytes), nil
}
