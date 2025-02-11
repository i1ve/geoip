package plaintext

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
	"os"
	"path/filepath"

	"geoip/lib"
)

var (
	defaultOutputDirForTextOut                  = filepath.Join("./", "output", "text")
	defaultOutputDirForClashRuleSetClassicalOut = filepath.Join("./", "output", "clash", "classical")
	defaultOutputDirForClashRuleSetIPCIDROut    = filepath.Join("./", "output", "clash", "ipcidr")
	defaultOutputDirForSurgeRuleSetOut          = filepath.Join("./", "output", "surge")
)

type textOut struct {
	Type        string
	Action      lib.Action
	Description string
	OutputDir   string
	Want        []string
	OnlyIPType  lib.IPType
}

func newTextOut(iType string, action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
	var tmp struct {
		OutputDir  string     `json:"outputDir"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if tmp.OutputDir == "" {
		switch iType {
		case typeTextOut:
			tmp.OutputDir = defaultOutputDirForTextOut
		case typeClashRuleSetClassicalOut:
			tmp.OutputDir = defaultOutputDirForClashRuleSetClassicalOut
		case typeClashRuleSetIPCIDROut:
			tmp.OutputDir = defaultOutputDirForClashRuleSetIPCIDROut
		case typeSurgeRuleSetOut:
			tmp.OutputDir = defaultOutputDirForSurgeRuleSetOut
		}
	}

	return &textOut{
		Type:        iType,
		Action:      action,
		Description: descTextOut,
		OutputDir:   tmp.OutputDir,
		Want:        tmp.Want,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

func (t *textOut) marshalBytes(entry *lib.Entry) ([]byte, error) {
	var err error

	var entryCidr []string
	switch t.OnlyIPType {
	case lib.IPv4:
		entryCidr, err = entry.MarshalText(lib.IgnoreIPv6)
	case lib.IPv6:
		entryCidr, err = entry.MarshalText(lib.IgnoreIPv4)
	default:
		entryCidr, err = entry.MarshalText()
	}
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	switch t.Type {
	case typeTextOut:
		err = t.marshalBytesForTextOut(&buf, entryCidr)
	case typeClashRuleSetClassicalOut:
		err = t.marshalBytesForClashRuleSetClassicalOut(&buf, entryCidr)
	case typeClashRuleSetIPCIDROut:
		err = t.marshalBytesForClashRuleSetIPCIDROut(&buf, entryCidr)
	case typeSurgeRuleSetOut:
		err = t.marshalBytesForSurgeRuleSetOut(&buf, entryCidr)
	default:
		return nil, lib.ErrNotSupportedFormat
	}
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *textOut) marshalBytesForTextOut(buf *bytes.Buffer, entryCidr []string) error {
	for _, cidr := range entryCidr {
		buf.WriteString(cidr)
		buf.WriteString("\n")
	}
	return nil
}

func (t *textOut) marshalBytesForClashRuleSetClassicalOut(buf *bytes.Buffer, entryCidr []string) error {
	buf.WriteString("payload:\n")
	for _, cidr := range entryCidr {
		ip, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}
		if ip.To4() != nil {
			buf.WriteString("  - IP-CIDR,")
		} else {
			buf.WriteString("  - IP-CIDR6,")
		}
		buf.WriteString(cidr)
		buf.WriteString("\n")
	}

	return nil
}

func (t *textOut) marshalBytesForClashRuleSetIPCIDROut(buf *bytes.Buffer, entryCidr []string) error {
	buf.WriteString("payload:\n")
	for _, cidr := range entryCidr {
		buf.WriteString("  - '")
		buf.WriteString(cidr)
		buf.WriteString("'\n")
	}

	return nil
}

func (t *textOut) marshalBytesForSurgeRuleSetOut(buf *bytes.Buffer, entryCidr []string) error {
	for _, cidr := range entryCidr {
		ip, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}
		if ip.To4() != nil {
			buf.WriteString("IP-CIDR,")
		} else {
			buf.WriteString("IP-CIDR6,")
		}
		buf.WriteString(cidr)
		buf.WriteString("\n")
	}

	return nil
}

func (t *textOut) writeFile(filename string, data []byte) error {
	if err := os.MkdirAll(t.OutputDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(t.OutputDir, filename), data, 0644); err != nil {
		return err
	}

	log.Printf("✅ [%s] %s --> %s", t.Type, filename, t.OutputDir)

	return nil
}
