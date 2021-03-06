package common

import (
	"errors"
	"fmt"
	"log"

	"github.com/openshift/osde2e/pkg/config"
	"github.com/openshift/osde2e/pkg/metadata"
	"github.com/openshift/osde2e/pkg/osd"
	"github.com/openshift/osde2e/pkg/upgrade"
)

// ChooseVersions sets versions in cfg if not set based on defaults and upgrade options.
// If a release stream is set for an upgrade the previous available version is used and it's image is used for upgrade.
func ChooseVersions(cfg *config.Config, osd *osd.OSD) (err error) {
	// when defined, use set version
	if len(cfg.ClusterVersion) != 0 {
		err = nil
	} else if osd == nil {
		err = errors.New("osd must be setup when upgrading with release stream")
	} else if cfg.UpgradeImage == "" && cfg.UpgradeReleaseStream != "" {
		err = setupUpgradeVersion(cfg, osd)
	} else {
		err = setupVersion(cfg, osd, false)
	}

	// Set the versions in metadata. If upgrade hasn't been chosen, it should still be omitted from the end result.
	metadata.Instance.ClusterVersion = cfg.ClusterVersion
	metadata.Instance.UpgradeVersion = cfg.UpgradeReleaseName

	return err
}

// chooses between default version and nightly based on target versions.
func setupVersion(cfg *config.Config, osd *osd.OSD, isUpgrade bool) (err error) {
	if len(cfg.ClusterVersion) > 0 {
		return
	}
	if cfg.MajorTarget != 0 || cfg.MinorTarget != 0 {
		// don't require major to be set
		if cfg.MajorTarget == 0 {
			cfg.MajorTarget = -1
		}
		// look for the default release and install it for this OSD cluster.
		if cfg.ClusterVersion, err = osd.LatestVersion(cfg.MajorTarget, cfg.MinorTarget); err == nil {
			log.Printf("CLUSTER_VERSION not set but a TARGET is, running '%s'", cfg.ClusterVersion)
		}
	}

	if len(cfg.ClusterVersion) == 0 {
		if cfg.ClusterVersion, err = osd.DefaultVersion(); err == nil {
			log.Printf("CLUSTER_VERSION not set, using the current default '%s'", cfg.ClusterVersion)
		} else {
			return fmt.Errorf("Error finding default cluster version: %v", err)
		}
	}

	return
}

// chooses version based on optimal upgrade path
func setupUpgradeVersion(cfg *config.Config, osd *osd.OSD) (err error) {
	// Decide the version to install
	err = setupVersion(cfg, osd, true)
	if err != nil {
		return err
	}

	cfg.UpgradeReleaseName, cfg.UpgradeImage, err = upgrade.LatestRelease(cfg, cfg.UpgradeReleaseStream, true)
	if err != nil {
		return fmt.Errorf("couldn't get latest release from release-controller: %v", err)
	}

	clusterVersion, err := osd.OpenshiftVersionToSemver(cfg.ClusterVersion)
	if err != nil {
		log.Printf("error while parsing cluster version %s: %v", cfg.ClusterVersion, err)
		return err
	}

	upgradeVersion, err := osd.OpenshiftVersionToSemver(cfg.UpgradeReleaseName)
	if err != nil {
		log.Printf("error while parsing upgrade version %s: %v", cfg.UpgradeReleaseName, err)
		return err
	}

	if !clusterVersion.LessThan(upgradeVersion) {
		log.Printf("Cluster version is equal to or newer than the upgrade version. Looking up previous version...")
		if cfg.ClusterVersion, err = osd.PreviousVersion(cfg.UpgradeReleaseName); err != nil {
			return fmt.Errorf("failed retrieving previous version to '%s': %v", cfg.UpgradeReleaseName, err)
		}
	}

	// set upgrade image
	log.Printf("Selecting version '%s' to be able to upgrade to '%s' on release stream '%s'",
		cfg.ClusterVersion, cfg.UpgradeReleaseName, cfg.UpgradeReleaseStream)
	return
}
