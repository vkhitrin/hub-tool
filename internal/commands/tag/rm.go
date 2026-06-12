/*
   Copyright 2020 Docker Hub Tool authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package tag

import (
	"context"
	"fmt"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"

	"github.com/docker/hub-tool/internal/ansi"
	"github.com/docker/hub-tool/internal/commands/commandutil"
	"github.com/docker/hub-tool/pkg/hub"
	"github.com/pkg/errors"
)

const (
	rmName = "rm"
)

type rmOptions struct {
	force bool
}

func newRmCmd(streams command.Streams, hubClient *hub.Client, parent string) *cobra.Command {
	var opts rmOptions
	return commandutil.NewForceCommand(commandutil.CommandConfig{
		Use:    rmName + " [OPTIONS] REPOSITORY:TAG",
		Short:  "Delete a tag in a repository",
		Args:   cli.ExactArgs(1),
		Parent: parent,
		Name:   rmName,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runRm(cmd.Context(), streams, hubClient, opts, args[0])
			return commandutil.IgnoreCanceled(err)
		},
	}, &opts.force, "Force deletion of the tag")
}

func runRm(ctx context.Context, streams command.Streams, hubClient *hub.Client, opts rmOptions, image string) error {
	normRef, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return err
	}
	normRef = reference.TagNameOnly(normRef)
	ref, ok := normRef.(reference.NamedTagged)
	if !ok {
		return fmt.Errorf("invalid reference: tag must be specified")
	}

	if !opts.force {
		if _, err := fmt.Fprintln(streams.Out(), ansi.Warn(fmt.Sprintf(`WARNING: You are about to permanently delete image "%s:%s"`, reference.FamiliarName(ref), ref.Tag()))); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(streams.Out(), ansi.Warn("         This action is irreversible")); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(streams.Out(), ansi.Info("Are you sure you want to delete the image tagged %q from repository %q? [y/N] "), ref.Tag(), reference.FamiliarName(ref)); err != nil {
			return err
		}
	}
	err = commandutil.ConfirmUnlessForced(ctx, streams.In(), opts.force, commandutil.YesConfirmation(errors.New("deletion aborted")))
	if err != nil {
		return err
	}

	if err := hubClient.RemoveTag(reference.FamiliarName(ref), ref.Tag()); err != nil {
		return err
	}
	_, err = fmt.Fprintln(streams.Out(), "Deleted", image)
	return err
}
