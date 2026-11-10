package onprem

import "testing"

const linuxArp = `? (192.168.33.243) at 86:68:c8:05:3c:a6 [ether] on ops0
? (192.168.33.49) at <incomplete> on ops0
? (192.168.33.150) at 0a:4b:34:b2:6e:e4 [ether] on ops0
? (10.0.0.1) at 9e:e2:ef:00:f9:1e [ether] on eth0
`

const macosArp = `? (192.168.64.3) at 86:68:c8:5:3c:a6 on bridge100 ifscope [ethernet]
? (192.168.64.4) at a:4b:34:b2:6e:e4 on bridge100 ifscope [ethernet]
`

func TestIpFromArp(t *testing.T) {
	tests := []struct {
		name  string
		table string
		mac   string
		want  string
	}{
		{"linux, a zero in the middle of the mac", linuxArp, "86:68:c8:05:3c:a6", "192.168.33.243"},
		{"linux, a zero in the first octet", linuxArp, "0a:4b:34:b2:6e:e4", "192.168.33.150"},
		{"linux, an octet of two zeros", linuxArp, "9e:e2:ef:00:f9:1e", "10.0.0.1"},
		{"upper case", linuxArp, "86:68:C8:05:3C:A6", "192.168.33.243"},
		{"mac, zeros dropped", macosArp, "86:68:c8:05:3c:a6", "192.168.64.3"},
		{"mac, zero dropped in the first octet", macosArp, "0a:4b:34:b2:6e:e4", "192.168.64.4"},
		{"not there", linuxArp, "52:54:00:12:34:56", ""},
		{"empty table", "", "86:68:c8:05:3c:a6", ""},
	}
	for _, tt := range tests {
		got, err := ipFromArp(tt.table, tt.mac)
		if err != nil || got != tt.want {
			t.Errorf("%s: %q, %v; want %q", tt.name, got, err, tt.want)
		}
	}
}

func TestIpFromArpWithoutAMacIsAnError(t *testing.T) {
	if _, err := ipFromArp(linuxArp, ""); err == nil {
		t.Error("no error")
	}
}

func TestIpFromArpDoesNotTakeAPartOfAnotherMac(t *testing.T) {
	// 0a:4b:34:b2:6e:e4 contains a:4b:34:b2:6e:e4, which is what a mac with its zeros dropped looks like
	table := "? (10.0.0.9) at 1a:4b:34:b2:6e:e4 [ether] on ops0\n"
	if got, _ := ipFromArp(table, "0a:4b:34:b2:6e:e4"); got != "" {
		t.Errorf("%q", got)
	}
}

func TestFormatOctet(t *testing.T) {
	for in, want := range map[string]string{
		"d6:3f:9b:0f:0c:c8": "d6:3f:9b:0f:0c:c8",
		"d6:3f:9b:f:c:c8":   "d6:3f:9b:0f:0c:c8",
		"D6:3F:9B:F:C:C8":   "d6:3f:9b:0f:0c:c8",
		"9e:e2:ef:0:f9:1e":  "9e:e2:ef:00:f9:1e",
	} {
		if got, err := formatOctet(in); err != nil || got != want {
			t.Errorf("%q: %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestIpFromArpSkipsAnIncompleteEntry(t *testing.T) {
	table := "? (10.0.0.5) at <incomplete> on ops0\n? (10.0.0.6) at 0b:4b:34:b2:6e:e4 [ether] on ops0\n"
	if got, err := ipFromArp(table, "0b:4b:34:b2:6e:e4"); err != nil || got != "10.0.0.6" {
		t.Errorf("%q, %v", got, err)
	}
	if got, _ := ipFromArp(table, "de:d1:0b:fb:6b:77"); got != "" {
		t.Errorf("%q", got)
	}
}
