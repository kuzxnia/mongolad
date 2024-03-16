package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/kuzxnia/loadbot/cli/workload"
	"github.com/kuzxnia/loadbot/lbot"
	"github.com/kuzxnia/loadbot/lbot/proto"
	"github.com/kuzxnia/loadbot/lbot/resourcemanager"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

func New(version string, commit string, date string) *cobra.Command {
	cobra.EnableCommandSorting = false
	cmd := cobra.Command{
		Use:     "loadbot",
		Short:   "A command-line database workload driver ",
		Version: fmt.Sprintf("%s (commit: %s) (build date: %s)", version, commit, date),
	}
	cmd.AddCommand(provideAgentCommand())
	cmd.AddGroup(&AgentGroup)
	cmd.AddCommand(provideWorkloadCommands()...)
	cmd.AddGroup(&WorkloadGroup)
	cmd.AddCommand(provideOrchiestrationCommands()...)
	cmd.Root().CompletionOptions.HiddenDefaultCmd = true

	return &cmd
}

var (
	Conn                       *grpc.ClientConn
	DefaultProgressInterval, _ = time.ParseDuration("200ms")
)

var WorkloadGroup = cobra.Group{
	ID:    "workload",
	Title: "Workload Commands:",
}

const (
	WorkloadRootCommand = "workload"

	CommandStartWorkload    = "start"
	CommandStopWorkload     = "stop"
	CommandWatchWorkload    = "watch"
	CommandProgressWorkload = "progress"
	CommandConfigWorkload   = "config"

	// config args
	ConfigFile = "config-file"
	AgentUri   = "agent-uri"
	Interval   = "interval"
	StdIn      = "stdin"
)

func provideWorkloadCommands() []*cobra.Command {
	persistentPreRunE := func(cmd *cobra.Command, args []string) (err error) {
		f := cmd.Flags()
		agentUri, _ := f.GetString(AgentUri)
		Conn, err = grpc.Dial(agentUri, grpc.WithInsecure())
		// valiedate connection
		if err != nil {
			log.Fatal("Found errors trying to connect to loadbot-agent:", err)
			return
		}
		return
	}
	persistentPostRun := func(cmd *cobra.Command, args []string) {
		Conn.Close()
	}

	startCommand := cobra.Command{
		Use:               CommandStartWorkload,
		Short:             "Start workload",
		GroupID:           WorkloadGroup.ID,
		PersistentPreRunE: persistentPreRunE,
		PersistentPostRun: persistentPostRun,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			flags := cmd.Flags()

			progress, _ := flags.GetBool("progress")
			interval, _ := flags.GetDuration(Interval)

			if progress {
				request := proto.StartWithProgressRequest{
					RefreshInterval: interval.String(),
				}
				return workload.StartWorkloadWithProgress(Conn, &request)
			} else {
				// todo: switch to local model aka cli.StartRequest
				request := proto.StartRequest{
					Watch: false,
				}

				return workload.StartWorkload(Conn, &request)
			}
		},
	}

	startCommandFlags := startCommand.Flags()
	startCommandFlags.BoolP("progress", "p", false, "Show progress of stress test")
	startCommandFlags.DurationP(Interval, "i", DefaultProgressInterval, "Progress refresh interval")
	// todo: add parent command and inherit this flag
	startCommandFlags.StringP(AgentUri, "u", "127.0.0.1:1234", "loadbot agent uri (default: 127.0.0.1:1234)")

	stopCommand := cobra.Command{
		Use:               CommandStopWorkload,
		Short:             "Stop workload",
		GroupID:           WorkloadGroup.ID,
		PersistentPreRunE: persistentPreRunE,
		PersistentPostRun: persistentPostRun,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// todo: switch to local model aka cli.StartRequest
			request := proto.StopRequest{}
			// response model could have worlkload id?

			return workload.StopWorkload(Conn, &request)
		},
	}
	stopCommandFlags := stopCommand.Flags()
	stopCommandFlags.StringP(AgentUri, "u", "127.0.0.1:1234", "loadbot agent uri (default: 127.0.0.1:1234)")

	watchCommand := cobra.Command{
		Use:               CommandWatchWorkload,
		Short:             "Watch stress test",
		GroupID:           WorkloadGroup.ID,
		PersistentPreRunE: persistentPreRunE,
		PersistentPostRun: persistentPostRun,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// building parameters for stop
			// check for params

			// todo: switch to local model aka cli.StartRequest
			request := proto.WatchRequest{}
			// response model could have worlkload id?

			return workload.WatchWorkload(Conn, &request)
		},
	}
	watchCommandFlags := watchCommand.Flags()
	watchCommandFlags.StringP(AgentUri, "u", "127.0.0.1:1234", "loadbot agent uri (default: 127.0.0.1:1234)")

	progressCommand := cobra.Command{
		Use:               CommandProgressWorkload,
		Short:             "Watch workload progress",
		GroupID:           WorkloadGroup.ID,
		PersistentPreRunE: persistentPreRunE,
		PersistentPostRun: persistentPostRun,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			flags := cmd.Flags()
			interval, _ := flags.GetDuration(Interval)

			request := proto.ProgressRequest{
				RefreshInterval: interval.String(),
			}

			return workload.WorkloadProgress(Conn, &request)
		},
	}
	progressCommandFlags := progressCommand.Flags()
	progressCommandFlags.DurationP(Interval, "i", DefaultProgressInterval, "Progress refresh interval")
	progressCommandFlags.StringP(AgentUri, "u", "127.0.0.1:1234", "loadbot agent uri (default: 127.0.0.1:1234)")

	configCommand := cobra.Command{
		Use:     CommandConfigWorkload,
		Short:   "Config",
		GroupID: WorkloadGroup.ID,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			flags := cmd.Flags()
			configFile, _ := flags.GetString(ConfigFile)
			stdin, _ := flags.GetBool(StdIn)

			if configFile == "" && !stdin {
				return errors.New("You need to provide configuration from either " + ConfigFile + " or " + StdIn)
			}

			var parsedConfig *lbot.ConfigRequest
			if stdin {
				parsedConfig, err = lbot.ParseStdInConfig()
				if err != nil {
					return err
				}
			}

			if configFile != "" {
				parsedConfig, err = lbot.ParseConfigFile(configFile)
				if err != nil {
					return err
				}
			}

			return workload.SetConfigWorkload(Conn, parsedConfig)
		},
	}
	configCommandFlags := configCommand.Flags()
	configCommandFlags.StringP(ConfigFile, "f", "", "file with workload configuration")
	configCommandFlags.Bool(StdIn, false, "get workload configuration from stdin")
	configCommandFlags.StringP(AgentUri, "u", "127.0.0.1:1234", "loadbot agent uri (default: 127.0.0.1:1234)")

	return []*cobra.Command{&startCommand, &stopCommand, &configCommand, &progressCommand}
}

var AgentGroup = cobra.Group{
	ID:    "agent",
	Title: "Agent Commands:",
}

const (
	AgentStartCommand = "start-agent"

	// agent args
	AgentName                    = "name"
	AgentPort                    = "port"
	WatchConfigFileChanges       = "watch-config"
	MetricsExportUrl             = "metrics_export_url"
	MetricsExportIntervalSeconds = "metrics_export_interval_seconds"
	MetricsExportPort            = "metrics_export_port"
)

func provideAgentCommand() *cobra.Command {
	startAgentCommand := cobra.Command{
		Use:     AgentStartCommand,
		Short:   "Start agent",
		GroupID: AgentGroup.ID,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			flags := cmd.Flags()

			name, _ := flags.GetString(AgentName)
			port, _ := flags.GetString(AgentPort)
			watchConfigFileChanges, _ := flags.GetBool(WatchConfigFileChanges)
			metricsExportUrl, _ := flags.GetString(MetricsExportUrl)
			metricsExportIntervalSeconds, _ := flags.GetUint64(MetricsExportIntervalSeconds)
			metricsExportPort, _ := flags.GetString(MetricsExportPort)

			agentConfig := &lbot.AgentRequest{
				Name:                         name,
				Port:                         port,
				MetricsExportUrl:             metricsExportUrl,
				MetricsExportIntervalSeconds: metricsExportIntervalSeconds,
				MetricsExportPort:            metricsExportPort,
			}

			configFile, _ := flags.GetString(ConfigFile)
			stdin, _ := flags.GetBool(StdIn)

			return StartAgent(
				cmd.Context(), agentConfig, watchConfigFileChanges, stdin, configFile,
			)
		},
	}

	flags := startAgentCommand.Flags()
	flags.StringP(AgentName, "n", "", "Agent name")
	flags.StringP(ConfigFile, "f", "", "Config file for loadbot-agent")
	flags.Bool(StdIn, false, "Provide configuration from stdin.")
	flags.Bool(WatchConfigFileChanges, false, "Watch config file changes.")
	flags.StringP(AgentPort, "p", "", "Agent port")
	flags.String(MetricsExportUrl, "", "Prometheus export url used for pushing metrics")
	flags.Uint64(MetricsExportIntervalSeconds, 0, "Prometheus export push interval")
	flags.String(MetricsExportPort, "", "Expose metrics on port instead pushing to prometheus")

	return &startAgentCommand
}

const (
	CommandInstall   = "install"
	CommandUpgrade   = "upgrade"
	CommandUnInstall = "uninstall"
	CommandList      = "list"

	// if not set will install localy without k8s
	FlagSourceKubeconfig = "k8s-config"
	FlagSourceContext    = "k8s-context"
	FlagSourceNamespace  = "k8s-namespace"

	FlagHelmTimeout   = "helm-timeout"
	FlagHelmValues    = "helm-values"
	FlagHelmSet       = "helm-set"
	FlagHelmSetString = "helm-set-string"
	FlagHelmSetFile   = "helm-set-file"
)

func provideOrchiestrationCommands() []*cobra.Command {
	installationCommand := cobra.Command{
		Use:     CommandInstall + " <name>",
		Aliases: []string{"i"},
		Short:   "Install workload driver with helm charts on k8s or only with docker locally",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			flags := cmd.Flags()

			srcKubeconfigPath, _ := flags.GetString(FlagSourceKubeconfig)
			srcContext, _ := flags.GetString(FlagSourceContext)
			srcNS, _ := flags.GetString(FlagSourceNamespace)

			helmTimeout, _ := flags.GetDuration(FlagHelmTimeout)
			helmValues, _ := flags.GetStringSlice(FlagHelmValues)
			helmSet, _ := flags.GetStringSlice(FlagHelmSet)
			helmSetString, _ := flags.GetStringSlice(FlagHelmSetString)
			helmSetFile, _ := flags.GetStringSlice(FlagHelmSetFile)

			rsm := resourcemanager.ResourceManagerConfig{
				KubeconfigPath: srcKubeconfigPath,
				Context:        srcContext,
				Namespace:      srcNS,
				HelmTimeout:    helmTimeout,
			}

			request := resourcemanager.InstallRequest{
				ResourceManagerConfig: rsm,
        Name: args[0],
				HelmValuesFiles:       helmValues,
				HelmValues:            helmSet,
				HelmStringValues:      helmSetString,
				HelmFileValues:        helmSetFile,
			}

			return InstallResources(&request)
		},
	}

	flags := installationCommand.Flags()
	// flags
	flags.StringP(FlagSourceKubeconfig, "k", "", "path of the kubeconfig file of the source PVC")
	flags.StringP(FlagSourceContext, "c", "", "context in the kubeconfig file of the source PVC")
	flags.StringP(FlagSourceNamespace, "n", "", "namespace of the source PVC")

	flags.DurationP(FlagHelmTimeout, "t", 1*time.Minute, "install/uninstall timeout for helm releases")
	flags.StringSliceP(FlagHelmValues, "f", nil, "set additional Helm values by a YAML file or a URL (can specify multiple)")
	flags.StringSlice(FlagHelmSet, nil, "set additional Helm values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	flags.StringSlice(FlagHelmSetString, nil, "set additional Helm STRING values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	flags.StringSlice(FlagHelmSetFile, nil, "set additional Helm values from respective files specified via the command line (can specify multiple or separate values with commas: key1=path1,key2=path2)")

	upgradeCommand := cobra.Command{
		Use:   CommandUpgrade + " <name>",
		Short: "Upgrade workload driver with helm charts on k8s or only with docker locally",
		// handle args - this is name of workload
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			flags := cmd.Flags()

			srcKubeconfigPath, _ := flags.GetString(FlagSourceKubeconfig)
			srcContext, _ := flags.GetString(FlagSourceContext)
			srcNS, _ := flags.GetString(FlagSourceNamespace)

			helmTimeout, _ := flags.GetDuration(FlagHelmTimeout)
			helmValues, _ := flags.GetStringSlice(FlagHelmValues)
			helmSet, _ := flags.GetStringSlice(FlagHelmSet)
			helmSetString, _ := flags.GetStringSlice(FlagHelmSetString)
			helmSetFile, _ := flags.GetStringSlice(FlagHelmSetFile)

			rsm := resourcemanager.ResourceManagerConfig{
				KubeconfigPath: srcKubeconfigPath,
				Context:        srcContext,
				Namespace:      srcNS,
				HelmTimeout:    helmTimeout,
			}

			request := resourcemanager.UpgradeRequest{
				ResourceManagerConfig: rsm,
        Name: args[0],
				HelmValuesFiles:       helmValues,
				HelmValues:            helmSet,
				HelmStringValues:      helmSetString,
				HelmFileValues:        helmSetFile,
			}

			return UpgradeResources(&request)
		},
	}

	uflags := upgradeCommand.Flags()
	// flags
	uflags.StringP(FlagSourceKubeconfig, "k", "", "path of the kubeconfig file of the source PVC")
	uflags.StringP(FlagSourceContext, "c", "", "context in the kubeconfig file of the source PVC")
	uflags.StringP(FlagSourceNamespace, "n", "", "namespace of the source PVC")
	uflags.DurationP(FlagHelmTimeout, "t", 1*time.Minute, "install/uninstall timeout for helm releases")
	uflags.StringSliceP(FlagHelmValues, "f", nil, "set additional Helm values by a YAML file or a URL (can specify multiple)")
	uflags.StringSlice(FlagHelmSet, nil, "set additional Helm values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	uflags.StringSlice(FlagHelmSetString, nil, "set additional Helm STRING values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	uflags.StringSlice(FlagHelmSetFile, nil, "set additional Helm values from respective files specified via the command line (can specify multiple or separate values with commas: key1=path1,key2=path2)")

	unInstallationCommand := cobra.Command{
		// todo: where to keep configuration? there will be couple workloads at the same time
		Use:   CommandUnInstall,
		Short: "Uninstall workload driver",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			flags := cmd.Flags()

			srcKubeconfigPath, _ := flags.GetString(FlagSourceKubeconfig)
			srcContext, _ := flags.GetString(FlagSourceContext)
			srcNS, _ := flags.GetString(FlagSourceNamespace)
			helmTimeout, _ := flags.GetDuration(FlagHelmTimeout)

			rsm := resourcemanager.ResourceManagerConfig{
				KubeconfigPath: srcKubeconfigPath,
				Context:        srcContext,
				Namespace:      srcNS,
				HelmTimeout:    helmTimeout,
			}

			request := resourcemanager.UnInstallRequest{
				ResourceManagerConfig: rsm,
        Name: args[0],
			}

			return UnInstallResources(&request)
		},
	}
	unflags := unInstallationCommand.Flags()
	// flags
	unflags.StringP(FlagSourceKubeconfig, "k", "", "path of the kubeconfig file of the source PVC")
	unflags.StringP(FlagSourceContext, "c", "", "context in the kubeconfig file of the source PVC")
	unflags.StringP(FlagSourceNamespace, "n", "", "namespace of the source PVC")
	unflags.DurationP(FlagHelmTimeout, "t", 1*time.Minute, "install/uninstall timeout for helm releases")

	listCommand := cobra.Command{
		// todo: where to keep configuration? there will be couple workloads at the same time
		Use:   CommandList,
		Short: "List workloads",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			flags := cmd.Flags()

			srcKubeconfigPath, _ := flags.GetString(FlagSourceKubeconfig)
			srcContext, _ := flags.GetString(FlagSourceContext)
			srcNS, _ := flags.GetString(FlagSourceNamespace)
			helmTimeout, _ := flags.GetDuration(FlagHelmTimeout)

			rsm := resourcemanager.ResourceManagerConfig{
				KubeconfigPath: srcKubeconfigPath,
				Context:        srcContext,
				Namespace:      srcNS,
				HelmTimeout:    helmTimeout,
			}

			request := resourcemanager.ListRequest{
				ResourceManagerConfig: rsm,
			}

			return ListResources(&request)
		},
	}
	lflags := listCommand.Flags()
	// flags
	lflags.StringP(FlagSourceKubeconfig, "k", "", "path of the kubeconfig file of the source PVC")
	lflags.StringP(FlagSourceContext, "c", "", "context in the kubeconfig file of the source PVC")
	lflags.StringP(FlagSourceNamespace, "n", "", "namespace of the source PVC")
	lflags.DurationP(FlagHelmTimeout, "t", 1*time.Minute, "install/uninstall timeout for helm releases")

	return []*cobra.Command{&installationCommand, &unInstallationCommand}
}

// todo: generate complection
