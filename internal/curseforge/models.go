package curseforge

import "time"

type ProjectLinks struct {
	WebsiteUrl string `json:"websiteUrl"`
	WikiUrl    string `json:"wikiUrl"`
	IssuesUrl  string `json:"issuesUrl"`
	SourceUrl  string `json:"sourceUrl"`
}

type Category struct {
	Id               int       `json:"id"`
	GameId           int       `json:"gameId"`
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	Url              string    `json:"url"`
	IconUrl          string    `json:"iconUrl"`
	DateModified     time.Time `json:"dateModified"`
	IsClass          bool      `json:"isClass"`
	ClassId          int       `json:"classId"`
	ParentCategoryId int       `json:"parentCategoryId"`
	DisplayIndex     int       `json:"displayIndex"`
}

type Author struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Url  string `json:"url"`
}

type Logo struct {
	Id           int    `json:"id"`
	ProjectId    int    `json:"modId"`
	Url          string `json:"url"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	ThumbnailUrl string `json:"thumbnailUrl"`
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
	ProjectId int              `json:"modId"`
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
	GameVersionTypeId      int       `json:"gameVersionTypeId"`
}

type File struct {
	Id                   int                   `json:"id"`
	GameId               int                   `json:"gameId"`
	ProjectId            int                   `json:"modId"`
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
	DownloadUrl          string                `json:"downloadUrl"`
	GameVersions         []string              `json:"gameVersions"`
	SortableGameVersions []SortableGameVersion `json:"sortableGameVersions"`
	Dependencies         []Dependency          `json:"dependencies"`
	FileFingerprint      int                   `json:"fileFingerprint"`
	Fingerprint          int                   `json:"fingerprint"`
}

type Project struct {
	Id                 int          `json:"id"`
	GameId             int          `json:"gameId"`
	Name               string       `json:"name"`
	Slug               string       `json:"slug"`
	Links              ProjectLinks `json:"links"`
	Summary            string       `json:"summary"`
	DownloadCount      int          `json:"downloadCount"`
	PrimaryCategoryId  int          `json:"primaryCategoryId"`
	Categories         []Category   `json:"categories"`
	ClassId            int          `json:"classId"`
	Authors            []Author     `json:"authors"`
	Logo               Logo         `json:"logo"`
	MainFileId         int          `json:"mainFileId"`
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

type GameId int

const (
	Minecraft GameId = 432
)

type FingerprintResult struct {
	Matches   []File `json:"matches"`
	Unmatched []int  `json:"unmatched"`
}
