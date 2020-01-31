/*
*  Copyright (c) WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
*
*  WSO2 Inc. licenses this file to you under the Apache License,
*  Version 2.0 (the "License"); you may not use this file except
*  in compliance with the License.
*  You may obtain a copy of the License at
*
*    http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing,
* software distributed under the License is distributed on an
* "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
* KIND, either express or implied.  See the License for the
* specific language governing permissions and limitations
* under the License.
 */

package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/cbroglie/mustache"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wso2/product-apim-tooling/import-export-cli/utils"
)

const installOperatorCmdLiteral = "operator"

var flagApiOperatorFile string

//var flagRegistryHost string
//var flagUsername string
//var flagPassword string
//var flagBatchMod bool

// These types define authorization credentials for docker-config
// Credential represents a credential for a docker registry
type Credential struct {
	Auth     string `json:"auth"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Auth represents list of docker registries with credentials
type Auth struct {
	Auths map[string]Credential `json:"auths"`
}

// installOperatorCmd represents the installOperator command
var installOperatorCmd = &cobra.Command{
	Use:   installOperatorCmdLiteral,
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		utils.Logln(utils.LogPrefixInfo + installOperatorCmdLiteral + " called")

		isLocalInstallation := flagApiOperatorFile != ""
		if !isLocalInstallation {
			installOLM("0.13.0")
		}

		// OLM installation requires time to install before installing the WSO2 API Operator
		// Hence getting user inputs
		registryUrl, repository, username, password := readInputs()

		createDockerSecret(registryUrl, username, password)
		installApiOperator(isLocalInstallation)
		createControllerConfigs(repository, isLocalInstallation) //TODO: renuka have to configure repository

	},
}

// installOLM installs Operator Lifecycle Manager (OLM) with the given version
func installOLM(version string) {
	utils.Logln(utils.LogPrefixInfo + "Installing OLM")

	cmd := exec.Command(
		utils.Kubectl,
		utils.K8sApply,
		"-f",
		fmt.Sprintf(utils.OlmCrdUrlTemplate, version),
		"-f",
		fmt.Sprintf(utils.OlmOlmUrlTemplate, version),
	)

	output, err := cmd.Output()
	if err != nil {
		utils.HandleErrorAndExit("Error installing Operator-Hub OLM tool", err)
	}

	fmt.Println(string(output))
}

// installApiOperator installs WSO2 api-operator
func installApiOperator(isLocalInstallation bool) {
	utils.Logln(utils.LogPrefixInfo + "Installing API Operator")
	operatorFile := flagApiOperatorFile
	if !isLocalInstallation {
		operatorFile = utils.OperatorYamlUrl
	}

	// Install the operator by running the following command
	cmd := exec.Command(
		utils.Kubectl,
		utils.K8sApply,
		"-f",
		operatorFile,
	)

	output, err := cmd.Output()
	if err != nil {
		utils.HandleErrorAndExit("Error installing WSO2 api-operator", err)
	}

	fmt.Println(string(output))
}

// createDockerSecret creates K8S secret with credentials for docker registry
func createDockerSecret(registryUrl string, username string, password string) {
	encodedCredential := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	auth := Auth{Auths: map[string]Credential{registryUrl: Credential{
		Auth:     encodedCredential,
		Username: username,
		Password: password,
	}}}

	authJsonByte, err := json.Marshal(auth)
	if err != nil {
		utils.HandleErrorAndExit("Error marshalling docker secret credentials ", err)
	}

	// write config-map to a temp file
	tempFile, err := ioutil.TempFile(os.TempDir(), "docker-secret-*.json")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write(authJsonByte); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	// Close the file
	if err := tempFile.Close(); err != nil {
		log.Fatal(err)
	}

	// get yaml output of k8s secret for accessing registry
	cmdCreate := exec.Command(
		utils.Kubectl,
		utils.Create,
		utils.K8sSecret,
		utils.K8sSecretTypeGeneric,
		utils.K8sDockerSecretName,
		fmt.Sprintf("--from-file=%s=%s", utils.K8sDockerSecretKeyName, tempFile.Name()),
		"--dry-run",
		"-o", "yaml",
	)

	output, err := cmdCreate.Output()
	if err != nil {
		utils.HandleErrorAndExit("Error rendering k8s secret for registry credentials", err)
	}

	// execute kubernetes command to create secret for accessing registry
	cmd := exec.Command(
		utils.Kubectl,
		utils.K8sApply,
		"-f",
		"-",
	)

	pipe, err := cmd.StdinPipe()
	pipe.Write(output)
	pipe.Close()

	output, err = cmd.Output()

	if err != nil {
		utils.HandleErrorAndExit("Error creating k8s secret for registry credentials", err)
	}

	fmt.Println(string(output))
}

// createControllerConfigs downloads the mustache, replaces repository value and creates the config: `controller-config`
func createControllerConfigs(repository string, isLocalInstallation bool) {
	utils.Logln(utils.LogPrefixInfo + "Installing controller configs")
	var mustacheTemplate string

	if !isLocalInstallation {
		// read from GitHub
		mustacheGistUrl := `https://gist.githubusercontent.com/renuka-fernando/6d6c64c786e6d13742e802534de3da4e/raw/d6191bc60f3bae659749e9db5f882bef6d1d062a/controller_conf.yaml`

		templateBytes, err := utils.ReadFromUrl(mustacheGistUrl)
		if err != nil {
			utils.HandleErrorAndExit("Error reading controller-configs from server", err)
		}
		mustacheTemplate = string(templateBytes)
	} else {
		// read from local file
		// TODO: renuka read from file
		mustacheTemplate = ""
	}

	k8sConfigMap, err := mustache.Render(mustacheTemplate, map[string]string{
		"usernameDockerRegistry": repository,
	})
	if err != nil {
		utils.HandleErrorAndExit("Error rendering controller-configs", err)
	}

	// execute kubernetes command to create secret for accessing registry
	cmd := exec.Command(
		utils.Kubectl,
		utils.K8sApply,
		"-f",
		"-",
	)

	pipe, err := cmd.StdinPipe()
	pipe.Write([]byte(k8sConfigMap))
	pipe.Close()

	output, err := cmd.Output()

	if err != nil {
		utils.HandleErrorAndExit("Error creating controller configs", err)
	}

	fmt.Println(string(output))
}

// readInputs reads docker-registry URL, repository, username and password from the user
func readInputs() (string, string, string, string) {
	isConfirm := false
	registryUrl := ""
	repository := "renukafernando-test"
	username := ""
	password := ""
	var err error

	for !isConfirm {
		registryUrl, err = utils.ReadInputString("Enter Docker-Registry URL", utils.DockerRegistryUrl, utils.UrlValidationRegex, true)
		if err != nil {
			utils.HandleErrorAndExit("Error reading Docker-Registry URL", err)
		}

		username, err = utils.ReadInputString("Enter Username", "", utils.UsernameValidationRegex, true)
		if err != nil {
			utils.HandleErrorAndExit("Error reading Username", err)
		}

		password, err = utils.ReadPassword("Enter Password")
		if err != nil {
			utils.HandleErrorAndExit("Error reading Password", err)
		}

		fmt.Println("")
		fmt.Println("Docker-Registry URL: " + registryUrl)
		fmt.Println("Repository         : " + repository)
		fmt.Println("Username           : " + username)

		isConfirmStr, err := utils.ReadInputString("Confirm configurations", "Y", "", false)
		if err != nil {
			utils.HandleErrorAndExit("Error reading user input Confirmation", err)
		}

		isConfirmStr = strings.ToUpper(isConfirmStr)
		isConfirm = isConfirmStr == "Y" || isConfirmStr == "YES"
	}

	return registryUrl, repository, username, password
}

// init using Cobra
func init() {
	installCmd.AddCommand(installOperatorCmd)
	installOperatorCmd.Flags().StringVarP(&flagApiOperatorFile, "from-file", "f", "", "Path to API Operator directory")
	//installOperatorCmd.Flags().StringVarP(&flagRegistryHost, "registry-host", "h", "", "URL of the registry host")
	//installOperatorCmd.Flags().StringVarP(&flagUsername, "username", "u", "", "Username for the registry repository")
	//installOperatorCmd.Flags().StringVarP(&flagPassword, "password", "p", "", "Password for the registry repository user")
	//installOperatorCmd.Flags().BoolVarP(&flagBatchMod, "batch-mod", "B", false, "Run in non-interactive (batch) mode")
}
