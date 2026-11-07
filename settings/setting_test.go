package settings

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestAppConfigValidate(t *testing.T) {
	tests := []struct {
		name         string
		grpcPort     int
		httpPort     int
		mysqlPort    int
		missingMySQL bool
		wantErr      bool
	}{
		{name: "valid", grpcPort: 8090, httpPort: 8091, mysqlPort: 3306},
		{name: "lower and upper boundaries", grpcPort: 1, httpPort: 65535, mysqlPort: 1},
		{name: "upper and lower boundaries", grpcPort: 65535, httpPort: 1, mysqlPort: 65535},
		{name: "missing mysql", grpcPort: 8090, httpPort: 8091, missingMySQL: true, wantErr: true},
		{name: "zero grpc port", grpcPort: 0, httpPort: 8091, mysqlPort: 3306, wantErr: true},
		{name: "negative grpc port", grpcPort: -1, httpPort: 8091, mysqlPort: 3306, wantErr: true},
		{name: "grpc port too large", grpcPort: 65536, httpPort: 8091, mysqlPort: 3306, wantErr: true},
		{name: "zero http port", grpcPort: 8090, httpPort: 0, mysqlPort: 3306, wantErr: true},
		{name: "negative http port", grpcPort: 8090, httpPort: -1, mysqlPort: 3306, wantErr: true},
		{name: "http port too large", grpcPort: 8090, httpPort: 65536, mysqlPort: 3306, wantErr: true},
		{name: "same listening ports", grpcPort: 8090, httpPort: 8090, mysqlPort: 3306, wantErr: true},
		{name: "zero mysql port", grpcPort: 8090, httpPort: 8091, mysqlPort: 0, wantErr: true},
		{name: "negative mysql port", grpcPort: 8090, httpPort: 8091, mysqlPort: -1, wantErr: true},
		{name: "mysql port too large", grpcPort: 8090, httpPort: 8091, mysqlPort: 65536, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &AppConfig{
				Port:        tt.grpcPort,
				HTTPPort:    tt.httpPort,
				MySQLConfig: &MySQLConfig{Port: tt.mysqlPort},
			}
			if tt.missingMySQL {
				cfg.MySQLConfig = nil
			}

			if err := cfg.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, want error = %v", err, tt.wantErr)
			}
		})
	}
}

func TestInitRejectsMissingMySQLConfig(t *testing.T) {
	originalConf := Conf
	viper.Reset()
	Conf = new(AppConfig)
	t.Cleanup(func() {
		viper.Reset()
		Conf = originalConf
	})
	t.Chdir(t.TempDir())

	config := []byte("name: test\nport: 8090\nhttp_port: 8091\n")
	if err := os.WriteFile("config.yaml", config, 0600); err != nil {
		t.Fatal(err)
	}

	err := Init()
	if err == nil {
		t.Fatal("Init accepted a configuration without the mysql section")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "mysql") {
		t.Fatalf("expected an error about missing MySQL configuration, got %v", err)
	}
}
