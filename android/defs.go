// Copyright 2015 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package android

import (
	"strings"
	"testing"

	"github.com/google/blueprint"
	_ "github.com/google/blueprint/bootstrap"
	"github.com/google/blueprint/proptools"
)

var (
	pctx = NewPackageContext("android/soong/android")

	cpPreserveSymlinks = pctx.VariableConfigMethod("cpPreserveSymlinks",
		Config.CpPreserveSymlinksFlags)

	// A phony rule that is not the built-in Ninja phony rule.  The built-in
	// phony rule has special behavior that is sometimes not desired.  See the
	// Ninja docs for more details.
	Phony = pctx.AndroidStaticRule("Phony",
		blueprint.RuleParams{
			Command:     "# phony $out",
			Description: "phony $out",
		})

	// GeneratedFile is a rule for indicating that a given file was generated
	// while running soong.  This allows the file to be cleaned up if it ever
	// stops being generated by soong.
	GeneratedFile = pctx.AndroidStaticRule("GeneratedFile",
		blueprint.RuleParams{
			Command:     "# generated $out",
			Description: "generated $out",
			Generator:   true,
		})

	// A copy rule.
	Cp = pctx.AndroidStaticRule("Cp",
		blueprint.RuleParams{
			Command:     "rm -f $out && cp $cpPreserveSymlinks $cpFlags $in $out",
			Description: "cp $out",
		},
		"cpFlags")

	CpExecutable = pctx.AndroidStaticRule("CpExecutable",
		blueprint.RuleParams{
			Command:     "rm -f $out && cp $cpPreserveSymlinks $cpFlags $in $out && chmod +x $out",
			Description: "cp $out",
		},
		"cpFlags")

	// A timestamp touch rule.
	Touch = pctx.AndroidStaticRule("Touch",
		blueprint.RuleParams{
			Command:     "touch $out",
			Description: "touch $out",
		})

	// A symlink rule.
	Symlink = pctx.AndroidStaticRule("Symlink",
		blueprint.RuleParams{
			Command:        "rm -f $out && ln -f -s $fromPath $out",
			Description:    "symlink $out",
			SymlinkOutputs: []string{"$out"},
		},
		"fromPath")

	ErrorRule = pctx.AndroidStaticRule("Error",
		blueprint.RuleParams{
			Command:     `echo "$error" && false`,
			Description: "error building $out",
		},
		"error")

	Cat = pctx.AndroidStaticRule("Cat",
		blueprint.RuleParams{
			Command:     "cat $in > $out",
			Description: "concatenate licenses $out",
		})

	// ubuntu 14.04 offcially use dash for /bin/sh, and its builtin echo command
	// doesn't support -e option. Therefore we force to use /bin/bash when writing out
	// content to file.
	writeFile = pctx.AndroidStaticRule("writeFile",
		blueprint.RuleParams{
			Command:     `/bin/bash -c 'echo -e "$$0" > $out' $content`,
			Description: "writing file $out",
		},
		"content")

	// Used only when USE_GOMA=true is set, to restrict non-goma jobs to the local parallelism value
	localPool = blueprint.NewBuiltinPool("local_pool")

	// Used only by RuleBuilder to identify remoteable rules. Does not actually get created in ninja.
	remotePool = blueprint.NewBuiltinPool("remote_pool")

	// Used for processes that need significant RAM to ensure there are not too many running in parallel.
	highmemPool = blueprint.NewBuiltinPool("highmem_pool")
)

func init() {
	pctx.Import("github.com/google/blueprint/bootstrap")
}

var (
	// echoEscaper escapes a string such that passing it to "echo -e" will produce the input value.
	echoEscaper = strings.NewReplacer(
		`\`, `\\`, // First escape existing backslashes so they aren't interpreted by `echo -e`.
		"\n", `\n`, // Then replace newlines with \n
	)

	// echoEscaper reverses echoEscaper.
	echoUnescaper = strings.NewReplacer(
		`\n`, "\n",
		`\\`, `\`,
	)

	// shellUnescaper reverses the replacer in proptools.ShellEscape
	shellUnescaper = strings.NewReplacer(`'\''`, `'`)
)

// WriteFileRule creates a ninja rule to write contents to a file.  The contents will be escaped
// so that the file contains exactly the contents passed to the function, plus a trailing newline.
func WriteFileRule(ctx BuilderContext, outputFile WritablePath, content string) {
	content = echoEscaper.Replace(content)
	content = proptools.ShellEscape(content)
	if content == "" {
		content = "''"
	}
	ctx.Build(pctx, BuildParams{
		Rule:        writeFile,
		Output:      outputFile,
		Description: "write " + outputFile.Base(),
		Args: map[string]string{
			"content": content,
		},
	})
}

// shellUnescape reverses proptools.ShellEscape
func shellUnescape(s string) string {
	// Remove leading and trailing quotes if present
	if len(s) >= 2 && s[0] == '\'' {
		s = s[1 : len(s)-1]
	}
	s = shellUnescaper.Replace(s)
	return s
}

// ContentFromFileRuleForTests returns the content that was passed to a WriteFileRule for use
// in tests.
func ContentFromFileRuleForTests(t *testing.T, params TestingBuildParams) string {
	t.Helper()
	if g, w := params.Rule, writeFile; g != w {
		t.Errorf("expected params.Rule to be %q, was %q", w, g)
		return ""
	}

	content := params.Args["content"]
	content = shellUnescape(content)
	content = echoUnescaper.Replace(content)

	return content
}
