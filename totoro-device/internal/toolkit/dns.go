package toolkit

import (
	"fmt"
	"net"
)

// DNSRecord DNS记录
type DNSRecord struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

// DNSQuery 执行DNS查询
func DNSQuery(query, queryType, server string) ([]DNSRecord, error) {
	if server == "" {
		server = "8.8.8.8:53"
	} else {
		if _, _, err := net.SplitHostPort(server); err != nil {
			server = server + ":53"
		}
	}

	records := []DNSRecord{}

	switch queryType {
	case "A", "AAAA":
		ips, err := net.LookupIP(query)
		if err != nil {
			return nil, err
		}

		for _, ip := range ips {
			recordType := "A"
			if ip.To4() == nil {
				recordType = "AAAA"
			}

			if queryType == "" || queryType == recordType {
				records = append(records, DNSRecord{
					Name:  query,
					Type:  recordType,
					Value: ip.String(),
					TTL:   3600,
				})
			}
		}

	case "PTR":
		names, err := net.LookupAddr(query)
		if err != nil {
			return nil, err
		}

		for _, name := range names {
			records = append(records, DNSRecord{
				Name:  query,
				Type:  "PTR",
				Value: name,
				TTL:   3600,
			})
		}

	case "CNAME":
		cname, err := net.LookupCNAME(query)
		if err != nil {
			return nil, err
		}

		records = append(records, DNSRecord{
			Name:  query,
			Type:  "CNAME",
			Value: cname,
			TTL:   3600,
		})

	case "MX":
		mxs, err := net.LookupMX(query)
		if err != nil {
			return nil, err
		}

		for _, mx := range mxs {
			records = append(records, DNSRecord{
				Name:  query,
				Type:  "MX",
				Value: fmt.Sprintf("%d %s", mx.Pref, mx.Host),
				TTL:   3600,
			})
		}

	case "TXT":
		txts, err := net.LookupTXT(query)
		if err != nil {
			return nil, err
		}

		for _, txt := range txts {
			records = append(records, DNSRecord{
				Name:  query,
				Type:  "TXT",
				Value: txt,
				TTL:   3600,
			})
		}

	default:
		// 默认查询A记录
		return DNSQuery(query, "A", server)
	}

	return records, nil
}

