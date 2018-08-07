package main

import (
	"context"
	"runtime"

	stub "github.com/NicolasT/zenko-operator/pkg/stub"
	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	k8sutil "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	"github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

        helm "k8s.io/helm/pkg/helm"
        helmVersion "k8s.io/helm/pkg/version"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
        logrus.Infof("Helm Version: %s", helmVersion.GetVersion())
}

func main() {
	printVersion()

	sdk.ExposeMetricsPort()

        options := []helm.Option{helm.Host("localhost:44134")}
        client := helm.NewClient(options...)

	resource := "zenko.io/v1alpha1"
	kind := "Zenko"
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("Failed to get watch namespace: %v", err)
	}
	resyncPeriod := 5
	logrus.Infof("Watching %s, %s, %s, %d", resource, kind, namespace, resyncPeriod)
	sdk.Watch(resource, kind, namespace, resyncPeriod)
	sdk.Handle(stub.NewHandler(client))
	sdk.Run(context.TODO())
}
