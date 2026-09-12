/*
 *    Copyright 2026 Han Li and contributors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package cmd

import (
	"context"

	"github.com/urfave/cli/v3"
)

func completeCommand(ctx context.Context, command *cli.Command) {
	// urfave/cli's subcommand completer reads parsed positional arguments,
	// which can omit partial flags or truncate at "-". A parentless view uses
	// the original os.Args instead, including the prefix being completed.
	completion := &cli.Command{
		Flags:    command.Flags,
		Commands: command.Commands,
		Writer:   command.Root().Writer,
	}
	cli.DefaultCompleteWithFlags(ctx, completion)
}
