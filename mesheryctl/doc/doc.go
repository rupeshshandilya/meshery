package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/layer5io/meshery/mesheryctl/internal/cli/root"
)

const markdownTemplateCommand = `---
layout: default
title: %s
permalink: %s
redirect_from: %s/
type: reference
display-title: "false"
language: en
command: %s
subcommand: %s
---

`

type cmdDoc struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Usage       string `yaml:"usage"`
	Example     string `yaml:"example"`
}

func prepender(filename string) string {
	file := strings.Split(filename, ".md")
	title := filepath.Base(file[0])
	words := strings.Split(title, "-")
	if len(words) <= 1 {
		url := "reference/" + words[0] + "/main"
		return fmt.Sprintf(markdownTemplateCommand, title, url, url, words[0], "nil")
	}
	if len(words) == 3 {
		url := "reference/" + words[0] + "/" + words[1] + "/" + words[2]
		return fmt.Sprintf(markdownTemplateCommand, title, url, url, words[1], words[2])
	}
	if len(words) == 4 {
		url := "reference/" + words[0] + "/" + words[1] + "/" + words[2] + "/" + words[3]
		return fmt.Sprintf(markdownTemplateCommand, title, url, url, words[1], words[2])
	}
	url := "reference/" + words[0] + "/" + words[1]
	return fmt.Sprintf(markdownTemplateCommand, title, url, url, words[1], "nil")
}

func linkHandler(name string) string {
	base := strings.TrimSuffix(name, path.Ext(name))
	words := strings.Split(base, "-")
	if len(words) <= 1 {
		return "/main"
	}
	if len(words) == 3 {
		return strings.ToLower(words[2])
	}
	if len(words) == 4 {
		return strings.ToLower(words[2]) + "/" + strings.ToLower(words[3])
	}
	return strings.ToLower(words[1])
}

func doc() {
	markDownPath := "../../docs/pages/reference/mesheryctl/" // Path for docs
	//yamlPath := "./internal/cli/root/testDoc/"

	fmt.Println("Scanning available commands...")
	cmd := root.TreePath() // Takes entire tree of mesheryctl commands

	// To skip the footer part "Auto generated by spf13/cobra.."
	cmd.DisableAutoGenTag = true

	fmt.Println("Generating markdown docs...")

	err := GenMarkdownTreeCustom(cmd, markDownPath, prepender, linkHandler)

	if err != nil {
		log.Fatal(err)
	}

	//fmt.Println("Generating yaml docs...")

	// Generates YAML for whole tree
	//err = GenYamlTreeCustom(cmd, markDownPath, subprepender, linkHandler)
	//if err != nil {
	//	log.Fatal(err)
	//}

	fmt.Println("Documentation generated at " + markDownPath)
}

func printOptions(buf *bytes.Buffer, cmd *cobra.Command, name string) error {
	flags := cmd.NonInheritedFlags()
	flags.SetOutput(buf)
	if flags.HasAvailableFlags() {
		buf.WriteString("## Options\n\n<pre class='codeblock-pre'>\n<div class='codeblock'>\n")
		flags.PrintDefaults()
		buf.WriteString("\n</div>\n</pre>\n\n")
	}

	parentFlags := cmd.InheritedFlags()
	parentFlags.SetOutput(buf)
	if parentFlags.HasAvailableFlags() {
		buf.WriteString("## Options inherited from parent commands\n\n<pre class='codeblock-pre'>\n<div class='codeblock'>\n")
		parentFlags.PrintDefaults()
		buf.WriteString("\n</div>\n</pre>\n\n")
	}
	return nil
}

// GenMarkdownCustom creates custom markdown output.
func GenMarkdownCustom(cmd *cobra.Command, w io.Writer, linkHandler func(string) string) error {
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	buf := new(bytes.Buffer)
	name := cmd.CommandPath()

	buf.WriteString("# " + name + "\n\n")
	buf.WriteString(cmd.Short + "\n\n")
	if len(cmd.Long) > 0 {
		buf.WriteString("## Synopsis\n\n")
		buf.WriteString(cmd.Long + "\n\n")
	}

	if cmd.Runnable() {
		buf.WriteString(fmt.Sprintf("<pre class='codeblock-pre'>\n<div class='codeblock'>\n%s\n\n</div>\n</pre> \n\n", cmd.UseLine()))
	}

	if len(cmd.Example) > 0 {
		buf.WriteString("## Examples\n\n")
		var examples = strings.Split(cmd.Example, "\n")
		for i := 0; i < len(examples); i++ {
			if examples[i] != "" && examples[i] != " " && examples[i] != "	" {
				if strings.HasPrefix(examples[i], "//") {
					buf.WriteString(strings.Replace(examples[i], "// ", "", -1) + "\n")
				} else {
					buf.WriteString(fmt.Sprintf("<pre class='codeblock-pre'>\n<div class='codeblock'>\n%s\n\n</div>\n</pre> \n\n", examples[i]))
				}
			}
		}
	}

	if err := printOptions(buf, cmd, name); err != nil {
		return err
	}
	if hasSeeAlso(cmd) {
		buf.WriteString("## See Also\n\n")
		if cmd.HasParent() {
			cmd.VisitParents(func(c *cobra.Command) {
				if c.DisableAutoGenTag {
					cmd.DisableAutoGenTag = c.DisableAutoGenTag
				}
			})
		}
		buf.WriteString("Go back to [command reference index](/reference/mesheryctl/) ")
		buf.WriteString("\n")
	}
	if !cmd.DisableAutoGenTag {
		buf.WriteString("###### Auto generated by spf13/cobra on " + time.Now().Format("2-Jan-2006") + "\n")
	}
	_, err := buf.WriteTo(w)
	return err
}

func hasSeeAlso(cmd *cobra.Command) bool {
	if cmd.HasParent() {
		return true
	}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		return true
	}
	return false
}

// Custom function to generate markdown docs with '-' as separator
func GenMarkdownTreeCustom(cmd *cobra.Command, dir string, filePrepender, linkHandler func(string) string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := GenMarkdownTreeCustom(c, dir, filePrepender, linkHandler); err != nil {
			return err
		}
	}

	basename := strings.Replace(cmd.CommandPath(), " ", "-", -1) + ".md"
	filename := filepath.Join(dir, basename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.WriteString(f, filePrepender(filename))
	if err != nil {
		return err
	}

	err = GenMarkdownCustom(cmd, f, linkHandler)
	if err != nil {
		return err
	}
	return nil
}

func GenYamlTreeCustom(cmd *cobra.Command, dir string, filePrepender, linkHandler func(string) string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := GenYamlTreeCustom(c, dir, filePrepender, linkHandler); err != nil {
			return err
		}
	}

	basename := "cmds.yml"
	filename := filepath.Join(dir, basename)
	f, err := os.OpenFile(basename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.WriteString(f, filePrepender(filename))
	if err != nil {
		return err
	}

	err = GenYamlCustom(cmd, f, linkHandler)
	if err != nil {
		return err
	}
	return nil
}

func GenYamlCustom(cmd *cobra.Command, w io.Writer, linkHandler func(string) string) error {
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	yamlDoc := cmdDoc{}

	yamlDoc.Name = cmd.CommandPath()
	yamlDoc.Description = cmd.Short
	yamlDoc.Usage = cmd.UseLine()
	if len(cmd.Example) > 0 {
		yamlDoc.Example = cmd.Example
	}

	fmt.Println(yamlDoc)
	final, err := yaml.Marshal(&yamlDoc)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	_, err = w.Write(final)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	doc()
}
