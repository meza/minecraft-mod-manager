package curseforge

import "time"

type ProjectLinks struct {
	WebsiteURL string `json:"websiteUrl"`
	WikiURL    string `json:"wikiUrl"`
	IssuesURL  string `json:"issuesUrl"`
	SourceURL  string `json:"sourceUrl"`
}

type Category struct {
	ID               int       `json:"id"`
	GameID           int       `json:"gameId"`
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	URL              string    `json:"url"`
	IconURL          string    `json:"iconUrl"`
	DateModified     time.Time `json:"dateModified"`
	IsClass          bool      `json:"isClass"`
	ClassID          int       `json:"classId"`
	ParentCategoryID int       `json:"parentCategoryId"`
	DisplayIndex     int       `json:"displayIndex"`
}

type Author struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Logo struct {
	ID           int    `json:"id"`
	ProjectID    int    `json:"modId"`
	URL          string `json:"url"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

type FileReleaseType int

const (
	Release FileReleaseType = 1
	Beta    FileReleaseType = 2
	Alpha   FileReleaseType = 3
)

type FileStatus int

const (
	Processing         FileStatus = 1
	ChangesRequired    FileStatus = 2
	UnderReview        FileStatus = 3
	Approved           FileStatus = 4
	Rejected           FileStatus = 5
	MalwareDetected    FileStatus = 6
	Deleted            FileStatus = 7
	Archived           FileStatus = 8
	Testing            FileStatus = 9
	Released           FileStatus = 10
	ReadyForReview     FileStatus = 11
	Deprecated         FileStatus = 12
	Baking             FileStatus = 13
	AwaitingPublishing FileStatus = 14
	FailedPublishing   FileStatus = 15
)

type FileHashAlgorithm int

const (
	SHA1 FileHashAlgorithm = 1
	MD5  FileHashAlgorithm = 2
)

type FileRelationType int

const (
	EmbeddedLibrary    FileRelationType = 1
	OptionalDependency FileRelationType = 2
	RequiredDependency FileRelationType = 3
	Tool               FileRelationType = 4
	Incompatible       FileRelationType = 5
	Include            FileRelationType = 6
)

type Dependency struct {
	ProjectID int              `json:"modId"`
	Type      FileRelationType `json:"type"`
}

type FileHash struct {
	Algorithm FileHashAlgorithm `json:"algo"`
	Hash      string            `json:"value"`
}

type SortableGameVersion struct {
	GameVersionName        string    `json:"gameVersionName"`
	GameVersion            string    `json:"gameVersion"`
	GameVersionPadded      string    `json:"gameVersionPadded"`
	GameVersionReleaseDate time.Time `json:"gameVersionReleaseDate"`
	GameVersionTypeID      int       `json:"gameVersionTypeId"`
}

type File struct {
	ID                   int                   `json:"id"`
	GameID               int                   `json:"gameId"`
	ProjectID            int                   `json:"modId"`
	IsAvailable          bool                  `json:"isAvailable"`
	DisplayName          string                `json:"displayName"`
	FileName             string                `json:"fileName"`
	ReleaseType          FileReleaseType       `json:"releaseType"`
	FileStatus           FileStatus            `json:"fileStatus"`
	Hashes               []FileHash            `json:"hashes"`
	FileDate             time.Time             `json:"fileDate"`
	FileLength           int                   `json:"fileLength"`
	DownloadCount        int                   `json:"downloadCount"`
	FileSizeOnDisk       int                   `json:"fileSizeOnDisk"`
	DownloadURL          string                `json:"downloadUrl"`
	GameVersions         []string              `json:"gameVersions"`
	SortableGameVersions []SortableGameVersion `json:"sortableGameVersions"`
	Dependencies         []Dependency          `json:"dependencies"`
	FileFingerprint      int                   `json:"fileFingerprint"`
	Fingerprint          int                   `json:"fingerprint"`
}

type Project struct {
	ID                 int          `json:"id"`
	GameID             int          `json:"gameId"`
	Name               string       `json:"name"`
	Slug               string       `json:"slug"`
	Links              ProjectLinks `json:"links"`
	Summary            string       `json:"summary"`
	DownloadCount      int          `json:"downloadCount"`
	PrimaryCategoryID  int          `json:"primaryCategoryId"`
	Categories         []Category   `json:"categories"`
	ClassID            int          `json:"classId"`
	Authors            []Author     `json:"authors"`
	Logo               Logo         `json:"logo"`
	MainFileID         int          `json:"mainFileId"`
	LatestFiles        []File       `json:"latestFiles"`
	DateCreated        time.Time    `json:"dateCreated"`
	DateModified       time.Time    `json:"dateModified"`
	DateReleased       time.Time    `json:"dateReleased"`
	GamePopularityRank int          `json:"gamePopularityRank"`
	ThumbsUpCount      int          `json:"thumbsUpCount"`
	Rating             int          `json:"rating"`
}

type Pagination struct {
	Cursor      int `json:"index"`
	PageSize    int `json:"pageSize"`
	ResultCount int `json:"resultCount"`
	TotalCount  int `json:"totalCount"`
}

type ModLoaderType int

const (
	Any        ModLoaderType = 0
	Forge      ModLoaderType = 1
	Cauldron   ModLoaderType = 2
	LiteLoader ModLoaderType = 3
	Fabric     ModLoaderType = 4
	Quilt      ModLoaderType = 5
	NeoForge   ModLoaderType = 6
)

type GameID int

const (
	Minecraft GameID = 432
)

type FingerprintResult struct {
	Matches   []File `json:"matches"`
	Unmatched []int  `json:"unmatched"`
}
