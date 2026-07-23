package cli

// Setup commands. Implemented in P2 (install/upgrade/doctor/uninstall) —
// see README.md "Phased build plan".

func runInstall(_ []string) int {
	return notImplemented("P2", "install",
		"scaffold tickets/, install the skill for detected agents, inject markers, write pickle.toml, register the first child-project")
}

func runUpgrade(_ []string) int {
	return notImplemented("P2", "upgrade",
		"refresh the installed skill payload + marker block to this binary's version")
}

func runDoctor(_ []string) int {
	return notImplemented("P2", "doctor",
		"verify install integrity: skill present, symlinks valid, markers present, every registered child path resolves")
}

func runUninstall(_ []string) int {
	return notImplemented("P2", "uninstall",
		"remove skill/symlinks/markers; leave tickets/ intact")
}
