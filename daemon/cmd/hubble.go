// Copyright 2020 Authors of Cilium
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

package cmd

import (
	"strings"

	"github.com/cilium/cilium/pkg/api"
	"github.com/cilium/cilium/pkg/hubble"
	"github.com/cilium/cilium/pkg/ipcache"
	"github.com/cilium/cilium/pkg/logging"
	"github.com/cilium/cilium/pkg/logging/logfields"
	"github.com/cilium/cilium/pkg/option"

	hubbleServe "github.com/cilium/hubble/cmd/serve"
	hubbleMetrics "github.com/cilium/hubble/pkg/metrics"
	"github.com/cilium/hubble/pkg/parser"
	hubbleServer "github.com/cilium/hubble/pkg/server"
	"github.com/cilium/hubble/pkg/server/serveroption"
	"github.com/sirupsen/logrus"
)

func (d *Daemon) launchHubble() {
	logger := logging.DefaultLogger.WithField(logfields.LogSubsys, "hubble")
	if !option.Config.EnableHubble {
		logger.Info("Hubble server is disabled")
		return
	}
	addresses := append(option.Config.HubbleListenAddresses, "unix://"+option.Config.HubbleSocketPath)
	for _, address := range addresses {
		// TODO: remove warning once mutual TLS has been implemented
		if !strings.HasPrefix(address, "unix://") {
			logger.WithField("address", address).Warn("Hubble server will be exposing its API insecurely on this address")
		}
	}
	payloadParser, err := parser.New(d, d, d, ipcache.IPIdentityCache, d)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize Hubble")
		return
	}
	s, err := hubbleServer.NewLocalServer(payloadParser, logger,
		serveroption.WithMaxFlows(option.Config.HubbleFlowBufferSize),
		serveroption.WithMonitorBuffer(option.Config.HubbleEventQueueSize),
		serveroption.WithCiliumDaemon(d))
	if err != nil {
		logger.WithError(err).Error("Failed to initialize Hubble")
		return
	}
	go s.Start()
	d.monitorAgent.GetMonitor().RegisterNewListener(d.ctx, hubble.NewHubbleListener(s))

	listeners, err := hubbleServe.SetupListeners(addresses, api.CiliumGroupName)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize Hubble")
		return
	}

	logger.WithField("addresses", addresses).Info("Starting Hubble server")
	if err := hubbleServe.Serve(d.ctx, logger, listeners, s); err != nil {
		logger.WithError(err).Error("Failed to start Hubble server")
		return
	}
	if option.Config.HubbleMetricsServer != "" {
		logger.WithFields(logrus.Fields{
			"address": option.Config.HubbleMetricsServer,
			"metrics": option.Config.HubbleMetrics,
		}).Info("Starting Hubble Metrics server")
		if err := hubbleMetrics.EnableMetrics(log, option.Config.HubbleMetricsServer, option.Config.HubbleMetrics); err != nil {
			logger.WithError(err).Warn("Failed to initialize Hubble metrics server")
			return
		}
	}
}
