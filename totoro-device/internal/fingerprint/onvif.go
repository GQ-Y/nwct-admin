package fingerprint

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ONVIFDeviceInfo struct {
	Manufacturer    string `json:"manufacturer"`
	Model           string `json:"model"`
	FirmwareVersion string `json:"firmware_version"`
	SerialNumber    string `json:"serial_number"`
	HardwareId      string `json:"hardware_id"`
}

// ONVIFGetDeviceInformation 访问 device service 的 GetDeviceInformation（部分设备可能需要认证）。
func ONVIFGetDeviceInformation(ctx context.Context, xaddr string) (*ONVIFDeviceInfo, error) {
	u := strings.TrimSpace(xaddr)
	if u == "" {
		return nil, fmt.Errorf("xaddr 为空")
	}
	// 只做超轻量探测，避免卡住扫描
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
	}

	soap := `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope"
 xmlns:tds="http://www.onvif.org/ver10/device/wsdl">
 <s:Body>
  <tds:GetDeviceInformation/>
 </s:Body>
</s:Envelope>`

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(soap))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")

	cli := &http.Client{Timeout: 3 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("ONVIF 需要认证: %s", resp.Status)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ONVIF 请求失败: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	info := parseONVIFDeviceInformation(b)
	if info == nil {
		return nil, fmt.Errorf("解析 ONVIF 响应失败")
	}
	return info, nil
}

type onvifEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetDeviceInformationResponse struct {
			Manufacturer    string `xml:"Manufacturer"`
			Model           string `xml:"Model"`
			FirmwareVersion string `xml:"FirmwareVersion"`
			SerialNumber    string `xml:"SerialNumber"`
			HardwareId      string `xml:"HardwareId"`
		} `xml:"GetDeviceInformationResponse"`
	} `xml:"Body"`
}

func parseONVIFDeviceInformation(b []byte) *ONVIFDeviceInfo {
	p := bytes.TrimSpace(b)
	if len(p) == 0 {
		return nil
	}
	var env onvifEnvelope
	if err := xml.Unmarshal(p, &env); err != nil {
		return nil
	}
	r := env.Body.GetDeviceInformationResponse
	if strings.TrimSpace(r.Manufacturer) == "" && strings.TrimSpace(r.Model) == "" {
		return nil
	}
	return &ONVIFDeviceInfo{
		Manufacturer:    strings.TrimSpace(r.Manufacturer),
		Model:           strings.TrimSpace(r.Model),
		FirmwareVersion: strings.TrimSpace(r.FirmwareVersion),
		SerialNumber:    strings.TrimSpace(r.SerialNumber),
		HardwareId:      strings.TrimSpace(r.HardwareId),
	}
}


