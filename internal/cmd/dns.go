package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/builtbyrobben/namecheap-cli/internal/namecheap"
	"github.com/builtbyrobben/namecheap-cli/internal/outfmt"
)

type DNSCmd struct {
	List        DNSListCmd        `cmd:"" help:"List DNS records for a domain"`
	Set         DNSSetCmd         `cmd:"" help:"Set DNS records for a domain"`
	Nameservers DNSNameserversCmd `cmd:"" help:"Get or set domain nameservers"`
	SetCustom   DNSSetCustomCmd   `cmd:"set-custom" help:"Set custom nameservers for a domain"`
}

type DNSNameserversCmd struct {
	Get DNSNameserversGetCmd `cmd:"" help:"Get nameservers for a domain"`
	Set DNSSetCustomCmd      `cmd:"" help:"Set custom nameservers for a domain"`
}

type DNSNameserversGetCmd struct {
	SLD string `arg:"" required:"" help:"Second-level domain (e.g., example)"`
	TLD string `arg:"" required:"" help:"Top-level domain (e.g., com)"`
}

func (cmd *DNSNameserversGetCmd) Run(ctx context.Context, flags *RootFlags) error {
	client, err := getNamecheapClient(flags.Sandbox)
	if err != nil {
		return err
	}

	resp, err := client.DNSGetList(ctx, cmd.SLD, cmd.TLD)
	if err != nil {
		return err
	}

	result := resp.CommandResponse.DNSNameservers

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"DOMAIN", "IS_USING_OUR_DNS", "NAMESERVERS"}
		rows := [][]string{{result.Domain, result.IsUsingOurDNS, strings.Join(result.Nameservers, ",")}}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	fmt.Fprintf(os.Stderr, "Nameservers for %s.%s\n\n", cmd.SLD, cmd.TLD)
	fmt.Printf("Using Namecheap DNS: %s\n", result.IsUsingOurDNS)
	for _, ns := range result.Nameservers {
		fmt.Printf("%s\n", ns)
	}

	return nil
}

type DNSListCmd struct {
	SLD string `arg:"" required:"" help:"Second-level domain (e.g., example)"`
	TLD string `arg:"" required:"" help:"Top-level domain (e.g., com)"`
}

func (cmd *DNSListCmd) Run(ctx context.Context, flags *RootFlags) error {
	client, err := getNamecheapClient(flags.Sandbox)
	if err != nil {
		return err
	}

	resp, err := client.DNSGetHosts(ctx, cmd.SLD, cmd.TLD)
	if err != nil {
		return err
	}

	hosts := resp.CommandResponse.DNSHosts

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, hosts)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"NAME", "TYPE", "ADDRESS", "TTL"}

		var rows [][]string
		for _, h := range hosts.Hosts {
			rows = append(rows, []string{h.Name, h.Type, h.Address, h.TTL})
		}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	if len(hosts.Hosts) == 0 {
		fmt.Fprintf(os.Stderr, "No DNS records found for %s.%s\n", cmd.SLD, cmd.TLD)
		return nil
	}

	fmt.Fprintf(os.Stderr, "DNS records for %s.%s\n\n", cmd.SLD, cmd.TLD)

	for _, h := range hosts.Hosts {
		fmt.Printf("%-20s  %-8s  %-40s  TTL: %s\n", h.Name, h.Type, h.Address, h.TTL)
	}

	return nil
}

type DNSSetCmd struct {
	SLD     string `arg:"" required:"" help:"Second-level domain (e.g., example)"`
	TLD     string `arg:"" required:"" help:"Top-level domain (e.g., com)"`
	Records string `required:"" help:"JSON array of records: [{\"host_name\":\"@\",\"record_type\":\"A\",\"address\":\"1.2.3.4\"}]"`
}

func (cmd *DNSSetCmd) Run(ctx context.Context, flags *RootFlags) error {
	var records []namecheap.DNSRecordInput

	if err := json.Unmarshal([]byte(cmd.Records), &records); err != nil {
		return fmt.Errorf("parse records JSON: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("at least one DNS record is required")
	}

	client, err := getNamecheapClient(flags.Sandbox)
	if err != nil {
		return err
	}

	resp, err := client.DNSSetHosts(ctx, cmd.SLD, cmd.TLD, records)
	if err != nil {
		return err
	}

	result := resp.CommandResponse.DNSSetResult

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"DOMAIN", "SUCCESS"}
		rows := [][]string{{result.Domain, result.IsSuccess}}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	if result.IsSuccess == trueString {
		fmt.Fprintf(os.Stderr, "DNS records updated for %s.%s\n", cmd.SLD, cmd.TLD)
	} else {
		fmt.Fprintf(os.Stderr, "Failed to update DNS records for %s.%s\n", cmd.SLD, cmd.TLD)
	}

	return nil
}

type DNSSetCustomCmd struct {
	SLD         string `arg:"" required:"" help:"Second-level domain (e.g., example)"`
	TLD         string `arg:"" required:"" help:"Top-level domain (e.g., com)"`
	Nameservers string `required:"" help:"Comma-separated list of nameservers (e.g., ns1.example.com,ns2.example.com)"`
}

func (cmd *DNSSetCustomCmd) Run(ctx context.Context, flags *RootFlags) error {
	if !flags.Force {
		return fmt.Errorf("refusing to change nameservers without --force")
	}

	client, err := getNamecheapClient(flags.Sandbox)
	if err != nil {
		return err
	}

	resp, err := client.DNSSetCustom(ctx, cmd.SLD, cmd.TLD, cmd.Nameservers)
	if err != nil {
		return err
	}

	result := resp.CommandResponse.DNSSetCustomResult

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if outfmt.IsPlain(ctx) {
		headers := []string{"DOMAIN", "UPDATED"}
		rows := [][]string{{result.Domain, result.Updated}}

		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	if result.Updated == trueString {
		fmt.Fprintf(os.Stderr, "Custom nameservers set for %s.%s\n", cmd.SLD, cmd.TLD)
	} else {
		return fmt.Errorf("failed to set custom nameservers for %s.%s", cmd.SLD, cmd.TLD)
	}

	return nil
}
