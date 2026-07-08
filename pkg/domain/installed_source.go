package domain

type InstalledSource struct {
	Profile  Profile
	FlakeRev FlakeRev
	Packages []ProfilePackage
	Files    []FileEntry
}
