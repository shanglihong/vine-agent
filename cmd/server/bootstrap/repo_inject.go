package bootstrap

import (
	"sync"
	"vine-agent/config"
	"vine-agent/domain/memory/profile"
	"vine-agent/infra/persistence/file"
	"vine-agent/infra/persistence/sqlite"
	"vine-agent/utils"
)

var (
	repoOnce      sync.Once
	repoContainer *RepoContainer
)

type RepoContainer struct {
	Session *sqlite.SessionStore
	Project *sqlite.ProjectStore
	User    *sqlite.UserStore

	Profile profile.ProfileRepository
}

func GetRepoContainer() *RepoContainer {
	repoOnce.Do(func() {
		repoContainer = newRepoContainer()
	})
	return repoContainer
}

func newRepoContainer() *RepoContainer {
	cfg := config.LoadConfig()
	dbPath := cfg.Storage.SQLiteDBPath
	fileDir := cfg.Storage.ProfileDir

	sessionStore, sessErr := sqlite.NewSessionStore(dbPath)
	utils.Panic(sessErr)
	projectStore, projectErr := sqlite.NewProjectStore(dbPath)
	utils.Panic(projectErr)
	userStore, userErr := sqlite.NewUserStore(dbPath)
	utils.Panic(userErr)
	profileStore := file.NewFileProfileRepository(fileDir)
	return &RepoContainer{
		Session: sessionStore,
		Project: projectStore,
		User:    userStore,
		Profile: profileStore,
	}
}
