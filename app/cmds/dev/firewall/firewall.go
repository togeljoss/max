package firewall

import (
	"fmt"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/crazy-max/WindowsSpyBlocker/app/dnsres"
	"github.com/crazy-max/WindowsSpyBlocker/app/menu"
	"github.com/crazy-max/WindowsSpyBlocker/app/utils/data"
	"github.com/crazy-max/WindowsSpyBlocker/app/utils/file"
	"github.com/crazy-max/WindowsSpyBlocker/app/utils/netu"
	"github.com/crazy-max/WindowsSpyBlocker/app/utils/pathu"
	"github.com/crazy-max/WindowsSpyBlocker/app/utils/print"
	"github.com/crazy-max/WindowsSpyBlocker/app/utils/timeu"
	"github.com/crazy-max/WindowsSpyBlocker/app/whois"
	"github.com/fatih/color"
)

// Menu of Firewall
func Menu(args ...string) (err error) {
	menuCommands := []menu.CommandOption{
		{
			Description: "Test IPs",
			Color:       color.FgHiYellow,
			Function:    testIps,
		},
	}

	menuOptions := menu.NewOptions("Firewall", "'menu' for help [dev-firewall]> ", 0, "")

	menuN := menu.NewMenu(menuCommands, menuOptions)
	menuN.Start()
	return
}

func testIps(args ...string) error {
	logsPath := path.Join(pathu.Logs)
	if err := file.CreateSubfolder(logsPath); err != nil {
		print.Error(err)
		return nil
	}

	defer timeu.Track(time.Now())
	testIpsByRule(data.RULES_EXTRA)
	testIpsByRule(data.RULES_SPY)
	testIpsByRule(data.RULES_UPDATE)

	fmt.Printf("\nLogs available in ")
	color.New(color.FgCyan).Printf("%s\n", strings.TrimLeft(logsPath, pathu.Current))

	return nil
}

func testIpsByRule(rule string) {
	fmt.Println()

	testCsv := path.Join(pathu.Logs, fmt.Sprintf("firewall-test-%s.csv", rule))

	fmt.Printf("Get IPs for %s... ", rule)
	fwIps, err := data.GetFirewallIpsByRule(rule)
	if err != nil {
		print.Error(err)
		return
	}
	print.Ok()

	testCsvFile, _ := os.Create(testCsv)
	testCsvFile.WriteString("IP,ORGANIZATION,COUNTRY,RESOLVED DATE,RESOLVED DOMAIN")
	for _, fwIp := range fwIps {
		if strings.Contains(fwIp.IP, "-") {
			testIpRange(fwIp.IP, testCsvFile)
		} else if netu.IsValidIPv4(fwIp.IP) {
			testIp(fwIp.IP, testCsvFile)
		}
	}

	testCsvFile.Sync()
	testCsvFile.Close()
	fmt.Println()
}

func testIpRange(ipRange string, testCsvFile *os.File) {
	ips, err := netu.GetIpsFromIPRange(ipRange)
	if err != nil {
		return
	}
	for _, ip := range ips {
		ipNet := net.ParseIP(ip)
		ipNet = ipNet.To4()
		if ipNet == nil {
			continue
		}
		//if ipNet[3] % 10 == 0 && ipNet[3] > 0 && ipNet[3] < 255 {
		if ipNet[3] > 0 && ipNet[3] < 255 {
			testIp(ip, testCsvFile)
		}
	}
}

func testIp(ip string, testCsvFile *os.File) {
	fmt.Print("\nTesting ")
	color.New(color.FgMagenta).Printf("%s", ip)
	fmt.Print("...\n")
	whoisResult := whois.GetWhois(ip)
	if whoisResult != (whois.Whois{}) {
		fmt.Print("  Organisation: ")
		color.New(color.FgCyan).Printf("%s\n", whoisResult.Org)
		fmt.Print("  Country: ")
		color.New(color.FgCyan).Printf("%s\n", whoisResult.Country)
		testCsvFile.WriteString(fmt.Sprintf("\n%s,%s,%s", ip, whoisResult.Org, whoisResult.Country))
	} else {
		return
	}
	dnsresList := dnsres.GetDnsRes(ip)
	if len(dnsresList) > 0 {
		countRes := 0
		fmt.Println("  Resolutions:")
		for _, res := range dnsresList {
			fmt.Printf("    %s - ", res.LastResolved.Format("2006-01-02"))
			color.New(color.FgCyan).Printf("%s\n", res.IpOrDomain)
			if countRes == 0 {
				testCsvFile.WriteString(fmt.Sprintf(",%s,%s", res.LastResolved.Format("2006-01-02"), res.IpOrDomain))
			} else {
				testCsvFile.WriteString(fmt.Sprintf("\n,,,%s,%s", res.LastResolved.Format("2006-01-02"), res.IpOrDomain))
			}
			countRes += 1
		}
	} else {
		testCsvFile.WriteString(",,")
	}
}
