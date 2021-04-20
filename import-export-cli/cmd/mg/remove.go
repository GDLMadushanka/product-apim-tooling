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
package mg

import (
	"github.com/spf13/cobra"
	"github.com/wso2/product-apim-tooling/import-export-cli/utils"
)

const (
	removeCmdLiteral   = "remove"
	removeCmdShortDesc = "Remove an environment for the Microgateway Adapter(s)"
	removeCmdLongDesc  = "Remove Environment and its configurations from the config file" +
		"for the Microgateway Adapter(s)."
)

const removeCmdExamples = utils.ProjectName + " " + removeCmdLiteral + " " +
	envCmdLiteral + " prod"

// RemoveCmd represents the remove command
var RemoveCmd = &cobra.Command{
	Use:     removeCmdLiteral,
	Short:   removeCmdShortDesc,
	Long:    removeCmdLongDesc,
	Example: removeCmdExamples,
	Run: func(cmd *cobra.Command, args []string) {
		utils.Logln(utils.LogPrefixInfo + removeCmdLiteral + " called")
	},
}

func init() {
	MgCmd.AddCommand(RemoveCmd)
}