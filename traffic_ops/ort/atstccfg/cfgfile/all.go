package cfgfile

import (
	"errors"
	"io"
	"math/rand"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-atscfg"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/traffic_ops/ort/atstccfg/config"
)

// GetAllConfigs returns a map[configFileName]configFileText
func GetAllConfigs(cfg config.TCCfg) ([]ATSConfigFile, error) {
	toData, err := GetTOData(cfg)
	if err != nil {
		return nil, errors.New("getting data from traffic ops: " + err.Error())
	}

	meta, err := GetMeta(toData)
	if err != nil {
		return nil, errors.New("creating meta: " + err.Error())
	}

	hasSSLMultiCertConfig := false
	configs := []ATSConfigFile{}
	for _, fi := range meta.ConfigFiles {
		txt, contentType, _, err := GetConfigFile(toData, fi)
		if err != nil {
			return nil, errors.New("getting config file '" + fi.APIURI + "': " + err.Error())
		}
		if fi.FileNameOnDisk == atscfg.SSLMultiCertConfigFileName {
			hasSSLMultiCertConfig = true
		}
		configs = append(configs, ATSConfigFile{ATSConfigMetaDataConfigFile: fi, Text: txt, ContentType: contentType})
	}

	if hasSSLMultiCertConfig {
		sslConfigs, err := GetSSLCertsAndKeyFiles(toData)
		if err != nil {
			return nil, errors.New("getting ssl key and cert config files: " + err.Error())
		}
		configs = append(configs, sslConfigs...)
	}

	return configs, nil
}

const HdrConfigFilePath = "Path"

// WriteConfigs writes the given configs as a RFC2046§5.1 MIME multipart/mixed message.
func WriteConfigs(configs []ATSConfigFile, output io.Writer) error {
	w := multipart.NewWriter(output)

	// Create a unique boundary. Because we're using a text encoding, we need to make sure the boundary text doesn't occur in any body.
	boundary := w.Boundary()
	randSet := `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`
	for _, cfg := range configs {
		for strings.Contains(cfg.Text, boundary) {
			boundary += string(randSet[rand.Intn(len(randSet))])
		}
	}
	if err := w.SetBoundary(boundary); err != nil {
		return errors.New("setting multipart writer boundary '" + boundary + "': " + err.Error())
	}

	io.WriteString(output, `MIME-Version: 1.0`+"\r\n"+`Content-Type: multipart/mixed; boundary="`+boundary+`"`+"\r\n\r\n")

	for _, cfg := range configs {
		hdr := map[string][]string{
			rfc.ContentType:   {cfg.ContentType},
			HdrConfigFilePath: []string{filepath.Join(cfg.Location, cfg.FileNameOnDisk)},
		}
		partW, err := w.CreatePart(hdr)
		if err != nil {
			return errors.New("creating multipart part for config file '" + cfg.FileNameOnDisk + "': " + err.Error())
		}
		if _, err := io.WriteString(partW, cfg.Text); err != nil {
			return errors.New("writing to multipart part for config file '" + cfg.FileNameOnDisk + "': " + err.Error())
		}
	}

	if err := w.Close(); err != nil {
		return errors.New("closing multipart writer and writing final boundary: " + err.Error())
	}
	return nil
}
