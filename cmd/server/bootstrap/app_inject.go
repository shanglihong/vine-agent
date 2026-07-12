package bootstrap

import (
	"sync"
	"vine-agent/app/agent"
	memoryapp "vine-agent/app/memory"
	projectapp "vine-agent/app/project"
	userapp "vine-agent/app/user"
	"vine-agent/domain/chat"
)

var (
	appOnce      sync.Once
	appContainer *AppContainer
)

type AppContainer struct {
	AgentService        agent.Service
	InteractionService  agent.InteractionService
	EvolutionAppService *memoryapp.EvolutionAppService
	SessionAppService   *memoryapp.SessionAppService
	UserAppService      *userapp.UserAppService
	ProjectAppService   *projectapp.ProjectAppService
}

func GetAppContainer(domainContainer *DomainContainer, chatModel chat.ChatModel) *AppContainer {
	domainOnce.Do(func() {
		appContainer = newAppContainer(domainContainer, chatModel)
	})
	return appContainer
}

func newAppContainer(domainContainer *DomainContainer, chatModel chat.ChatModel) *AppContainer {
	agentSvc := agent.NewService(chatModel, domainContainer.SessionService)
	interactionSvc := agent.NewInteractionService(agentSvc, domainContainer.SessionService)
	evolutionAppSvc := memoryapp.NewEvolutionAppService(domainContainer.SessionService, domainContainer.ProfileService, domainContainer.EvolutionService)
	userAppSvc := userapp.NewUserAppService(domainContainer.UserService, domainContainer.SessionService, domainContainer.ProfileService, evolutionAppSvc)
	projectAppSvc := projectapp.NewProjectAppService(domainContainer.ProjectService)
	sessionAppSvc := memoryapp.NewSessionAppService(domainContainer.SessionService, domainContainer.ProjectService)
	return &AppContainer{
		AgentService:        agentSvc,
		InteractionService:  interactionSvc,
		EvolutionAppService: evolutionAppSvc,
		SessionAppService:   sessionAppSvc,
		UserAppService:      userAppSvc,
		ProjectAppService:   projectAppSvc,
	}
}
