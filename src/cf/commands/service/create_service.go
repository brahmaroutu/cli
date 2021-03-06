package service

import (
	"cf/api"
	"cf/configuration"
	"cf/errors"
	"cf/models"
	"cf/requirements"
	"cf/terminal"
	"fmt"
	"github.com/codegangsta/cli"
)

type ServiceCreator interface {
	CreateService(serviceName string, planName string, serviceInstanceName string) (bool, error) // idempotent
}

type CreateService struct {
	ui          terminal.UI
	config      configuration.Reader
	serviceRepo api.ServiceRepository
}

func NewCreateService(ui terminal.UI, config configuration.Reader, serviceRepo api.ServiceRepository) (cmd CreateService) {
	cmd.ui = ui
	cmd.config = config
	cmd.serviceRepo = serviceRepo
	return
}

func (cmd CreateService) GetRequirements(reqFactory requirements.Factory, c *cli.Context) (reqs []requirements.Requirement, err error) {
	if len(c.Args()) != 3 {
		err = errors.New("Incorrect Usage")
		cmd.ui.FailWithUsage(c, "create-service")
		return
	}

	reqs = []requirements.Requirement{
		reqFactory.NewLoginRequirement(),
		reqFactory.NewTargetedSpaceRequirement(),
	}

	return
}

func (cmd CreateService) Run(c *cli.Context) {
	serviceName := c.Args()[0]
	planName := c.Args()[1]
	serviceInstanceName := c.Args()[2]

	cmd.ui.Say("Creating service %s in org %s / space %s as %s...",
		terminal.EntityNameColor(serviceInstanceName),
		terminal.EntityNameColor(cmd.config.OrganizationFields().Name),
		terminal.EntityNameColor(cmd.config.SpaceFields().Name),
		terminal.EntityNameColor(cmd.config.Username()),
	)

	err := cmd.CreateService(serviceName, planName, serviceInstanceName)

	switch err.(type) {
	case nil:
		cmd.ui.Ok()
	case *errors.ModelAlreadyExistsError:
		cmd.ui.Ok()
		cmd.ui.Warn(err.Error())
	default:
		cmd.ui.Failed(err.Error())
	}
}

func (cmd CreateService) CreateService(serviceName string, planName string, serviceInstanceName string) (apiErr error) {
	offerings, apiErr := cmd.serviceRepo.FindServiceOfferingsForSpaceByLabel(cmd.config.SpaceFields().Guid, serviceName)
	if apiErr != nil {
		return
	}
	plan, apiErr := findPlanFromOfferings(offerings, planName)
	if apiErr != nil {
		return
	}

	return cmd.serviceRepo.CreateServiceInstance(serviceInstanceName, plan.Guid)
}

func findOfferings(offerings []models.ServiceOffering, name string) (matchingOfferings models.ServiceOfferings, err error) {
	for _, offering := range offerings {
		if name == offering.Label {
			matchingOfferings = append(matchingOfferings, offering)
		}
	}

	if len(matchingOfferings) == 0 {
		err = errors.New(fmt.Sprintf("Could not find any offerings with name %s", name))
	}
	return
}

func findPlanFromOfferings(offerings models.ServiceOfferings, name string) (plan models.ServicePlanFields, err error) {
	for _, offering := range offerings {
		for _, plan := range offering.Plans {
			if name == plan.Name {
				return plan, nil
			}
		}
	}

	err = errors.New(fmt.Sprintf("Could not find plan with name %s", name))
	return
}
