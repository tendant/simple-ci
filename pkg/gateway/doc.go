// Package gateway provides a reusable CI Gateway library that can be embedded into other Go applications.
//
// # Overview
//
// The Simple CI Gateway is a stateless, provider-agnostic CI Gateway that exposes a clean REST API
// backed by Concourse CI (with support for additional providers planned).
//
// # Basic Usage
//
// Create a gateway programmatically:
//
//	cfg := &gateway.Config{
//		Server: gateway.ServerConfig{
//			Port:         8080,
//			ReadTimeout:  30 * time.Second,
//			WriteTimeout: 30 * time.Second,
//		},
//		Auth: gateway.AuthConfig{
//			APIKeys: []gateway.APIKey{
//				{Name: "my-app", Key: "secret-key-here"},
//			},
//		},
//		Provider: gateway.ProviderConfig{
//			Kind: "concourse",
//			Concourse: &gateway.ConcourseConfig{
//				URL:      "http://localhost:9001",
//				Team:     "main",
//				Username: "admin",
//				Password: "admin",
//			},
//		},
//		Jobs: jobs, // []*models.Job
//		Logging: gateway.LoggingConfig{
//			Level:  "info",
//			Format: "json",
//		},
//	}
//
//	gw, err := gateway.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
//	defer cancel()
//
//	if err := gw.Start(ctx); err != nil {
//		log.Fatal(err)
//	}
//
// # Using with Existing HTTP Server
//
// Integrate the gateway into an existing HTTP server:
//
//	gw, err := gateway.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Mount the gateway under a specific path
//	http.Handle("/ci/", http.StripPrefix("/ci", gw.Handler()))
//
//	// Add your own routes
//	http.HandleFunc("/custom", myHandler)
//
//	http.ListenAndServe(":8080", nil)
//
// # Environment-based Configuration
//
// Load configuration from environment variables (requires .env file):
//
//	gw, err := gateway.NewFromEnv("configs/jobs.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
//	defer cancel()
//
//	if err := gw.Start(ctx); err != nil {
//		log.Fatal(err)
//	}
//
// # Direct Service Access
//
// Access the service layer directly for programmatic control:
//
//	gw, err := gateway.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	svc := gw.Service()
//
//	// Trigger a run programmatically
//	run, err := svc.TriggerRun(ctx, "job_id", map[string]interface{}{
//		"git_sha": "abc123",
//	}, "")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Printf("Triggered run: %s\n", run.RunID)
package gateway
