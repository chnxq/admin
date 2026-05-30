// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package server

import (
	"crypto/tls"

	tlsutils "github.com/chnxq/x-utils/tls"
	conf "github.com/chnxq/xkitpkg/conf/v1"
)

func loadServerTLSConfig(cfg *conf.TLS) (*tls.Config, error) {
	if cfg == nil {
		return nil, nil
	}
	if cfg.GetFile() != nil {
		return tlsutils.LoadServerTlsConfigFile(
			cfg.GetFile().GetKeyPath(),
			cfg.GetFile().GetCertPath(),
			cfg.GetFile().GetCaPath(),
			cfg.GetInsecureSkipVerify(),
		)
	}
	if cfg.GetConfig() != nil {
		return tlsutils.LoadServerTlsConfigString(
			cfg.GetConfig().GetKeyPem(),
			cfg.GetConfig().GetCertPem(),
			cfg.GetConfig().GetCaPem(),
			cfg.GetInsecureSkipVerify(),
		)
	}
	return nil, nil
}

func loadClientTLSConfig(cfg *conf.TLS) (*tls.Config, error) {
	if cfg == nil {
		return nil, nil
	}
	if cfg.GetFile() != nil {
		return tlsutils.LoadClientTlsConfigFile(
			cfg.GetFile().GetKeyPath(),
			cfg.GetFile().GetCertPath(),
			cfg.GetFile().GetCaPath(),
		)
	}
	if cfg.GetConfig() != nil {
		return tlsutils.LoadClientTlsConfigString(
			cfg.GetConfig().GetKeyPem(),
			cfg.GetConfig().GetCertPem(),
			cfg.GetConfig().GetCaPem(),
		)
	}
	return nil, nil
}
