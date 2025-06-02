package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/joho/godotenv"
)

type KamalDeploy struct {
	ServerIPs []string
	Domain    string
	Port      string
	Secrets   []string
	Registry  string
	ImageName string
}

const (
	ServerIPVar = "YEETFILE_SERVER_IP_LIST"

	RegistryServerVar = "KAMAL_REGISTRY_SERVER"

	ConfigPath  = "./config"
	SecretsPath = "./.kamal"
)

var deployTemplate = template.Must(template.New("").Parse(`service: yeetfile
image: {{ .ImageName }}

servers:
  {{ range .ServerIPs }}- {{ . }}{{ end }}

registry:
  server: "{{ .Registry }}"
  username:
    - KAMAL_REGISTRY_USERNAME
  password:
    - KAMAL_REGISTRY_PASSWORD

env:
  secret:{{ range .Secrets }}
    - {{ . }}{{ end }}
{{ if or .Domain .Port }}
proxy:
  {{ if .Domain }}host: {{ .Domain }}
  ssl: true{{ end }}
  {{ if .Port }}app_port: {{ .Port }}{{ end }}
  forward_headers: true
  healthcheck:
    path: /up
    interval: 3
    timeout: 3
{{ end }}
`))

var secretsTemplate = template.Must(template.New("").Parse(`{{ range . }}{{ . }}=${{ . }}
{{ end }}
`))

func main() {
	var deployType string
	if len(os.Args) > 1 {
		deployType = os.Args[1]
	}

	envFile := fmt.Sprintf("%s.env", deployType)
	err := godotenv.Load(envFile)
	if err != nil {
		log.Fatalln(err)
	}

	serverVars, allVars := readEnvVars(envFile)
	deploy := createKamalStruct(envFile)
	deploy.Secrets = serverVars

	var deployBuf bytes.Buffer
	err = deployTemplate.Execute(&deployBuf, deploy)
	if err != nil {
		log.Fatalln(err)
	}

	var secretsBuf bytes.Buffer
	err = secretsTemplate.Execute(&secretsBuf, allVars)
	if err != nil {
		log.Fatalln(err)
	}

	writeKamalFiles(deployType, deployBuf.String(), secretsBuf.String())
}

func createKamalStruct(envFile string) KamalDeploy {
	var (
		domain    string
		port      string
		imageName string
		registry  string
	)

	serverIPList := os.Getenv(ServerIPVar)
	if len(serverIPList) == 0 {
		log.Fatalf("Missing %s in %s\n", ServerIPVar, envFile)
	}

	serverIPList = strings.ReplaceAll(serverIPList, " ", "")
	serverIPs := strings.Split(serverIPList, ",")
	if len(serverIPs) == 1 {
		domain = os.Getenv("YEETFILE_DOMAIN")
		domain = strings.ReplaceAll(domain, "http://", "")
		domain = strings.ReplaceAll(domain, "https://", "")
	}

	port = os.Getenv("YEETFILE_PORT")
	imageName = os.Getenv("YEETFILE_IMAGE_NAME")
	if len(imageName) == 0 {
		log.Fatalf("Missing YEETFILE_IMAGE_NAME in %s", envFile)
	}

	registry = os.Getenv(RegistryServerVar)
	if len(registry) == 0 {
		log.Fatalf("Missing %s in %s", RegistryServerVar, envFile)
	}

	return KamalDeploy{
		ServerIPs: serverIPs,
		Domain:    domain,
		Port:      port,
		Registry:  registry,
		ImageName: imageName,
	}
}

func writeKamalFiles(deployType, deployContents, secretsContents string) {
	_, configPathErr := os.Stat(ConfigPath)
	_, secretsPathErr := os.Stat(SecretsPath)
	if configPathErr != nil || secretsPathErr != nil {
		log.Fatalf("Missing '%s' and/or '%s' directories!\n" +
			"Ensure you are running this script from the root level of the " +
			"project directory.")
	}

	deployName := "deploy.yml"
	secretsName := "secrets"
	if len(deployType) > 0 {
		deployName = fmt.Sprintf("deploy.%s.yml", deployType)
		secretsName = fmt.Sprintf("secrets.%s", deployType)
	}

	deployPath := path.Join(ConfigPath, deployName)
	secretsPath := path.Join(SecretsPath, secretsName)
	err := os.WriteFile(deployPath, []byte(deployContents), 0644)
	if err != nil {
		log.Fatalf("Error writing deployment file: %v\n", err)
	}

	err = os.WriteFile(secretsPath, []byte(secretsContents), 0644)
	if err != nil {
		log.Fatalf("Error writing secrets file: %v\n", err)
	}

	fmt.Printf("Kamal files written:\n-- Config: %s\n-- Secrets: %s\n",
		deployPath,
		secretsPath)
}

func readEnvVars(envFile string) ([]string, []string) {
	var (
		serverVarList []string
		fullVarList   []string
	)

	file, err := os.Open(envFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		envVar := strings.Split(line, "=")
		varName := envVar[0]
		if varName == ServerIPVar {
			continue
		}

		if strings.HasPrefix(varName, "YEETFILE_") {
			serverVarList = append(serverVarList, varName)
			fullVarList = append(fullVarList, varName)
		} else if strings.HasPrefix(varName, "KAMAL_") {
			fullVarList = append(fullVarList, varName)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return serverVarList, fullVarList
}
