package bootstrap

import (
	"sync"
	"vine-agent/config"
	"vine-agent/domain/chat"
	"vine-agent/domain/memory/profile"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/project"
	"vine-agent/domain/user"
	"vine-agent/infra/extractor"
)

var (
	domainOnce      sync.Once
	domainContainer *DomainContainer
)

type DomainContainer struct {
	SessionService   session.SessionService
	EvolutionService profile.EvolutionService
	UserService      user.UserService
	ProjectService   project.ProjectService
	ProfileService   profile.ProfileService
}

func GetDomainContainer(repoContainer *RepoContainer, chatModel chat.ChatModel) *DomainContainer {
	domainOnce.Do(func() {
		domainContainer = newDomainContainer(repoContainer, chatModel)
	})
	return domainContainer
}

func newDomainContainer(repoContainer *RepoContainer, chatModel chat.ChatModel) *DomainContainer {
	cfg := config.LoadConfig()
	llmExtractor := extractor.NewLLMExtractor(chatModel)
	sessionSvc := session.NewSessionService(repoContainer.Session)
	evolutionSvc := profile.NewEvolutionService(llmExtractor)
	userDomainSvc := user.NewUserService(repoContainer.User)
	projectDomainSvc := project.NewProjectService(repoContainer.Project, sessionSvc, cfg.Storage.ProjectRootDir)
	profileService := profile.NewProfileService(repoContainer.Profile)
	return &DomainContainer{
		SessionService:   sessionSvc,
		EvolutionService: evolutionSvc,
		UserService:      userDomainSvc,
		ProjectService:   projectDomainSvc,
		ProfileService:   profileService,
	}
}
