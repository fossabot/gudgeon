package engine

import (
	"encoding/base64"
	"os"
	"runtime"
	"strings"

	"github.com/google/uuid"

	"github.com/chrisruffalo/gudgeon/config"
	gmetrics "github.com/chrisruffalo/gudgeon/metrics"
	"github.com/chrisruffalo/gudgeon/resolver"
	"github.com/chrisruffalo/gudgeon/rule"
	"github.com/chrisruffalo/gudgeon/util"
)

// returns an array of the GudgeonLists that are assigned either by name or by tag from within the list of GudgeonLists in the config file
func assignedLists(listNames []string, listTags []string, lists []*config.GudgeonList) []*config.GudgeonList {
	// empty list
	should := []*config.GudgeonList{}

	// check names
	for _, list := range lists {
		if util.StringIn(list.Name, listNames) {
			should = append(should, list)
			continue
		}

		for _, tag := range list.SafeTags() {
			if util.StringIn(tag, listTags) {
				should = append(should, list)
				break
			}
		}
	}

	return should
}

func New(conf *config.GudgeonConfig, metrics gmetrics.Metrics) (Engine, error) {
	// create return object
	engine := new(engine)
	engine.config = conf

	// create session key
	uuid := uuid.New()

	// and make a hidden session folder from  it
	engine.session = "." + base64.RawURLEncoding.EncodeToString([]byte(uuid.String()))

	// make required paths
	os.MkdirAll(conf.Home, os.ModePerm)
	os.MkdirAll(conf.SessionRoot(), os.ModePerm)
	os.MkdirAll(engine.Root(), os.ModePerm)

	// configure resolvers
	engine.resolvers = resolver.NewResolverMap(conf, conf.Resolvers)

	// get lists from the configuration
	lists := conf.Lists

	// empty groups list of size equal to available groups
	workingGroups := append([]*config.GudgeonGroup{}, conf.Groups...)

	// use length of working groups to make list of active groups
	groups := make([]*group, len(workingGroups))

	// process groups
	for idx, configGroup := range workingGroups {
		// create active group for gorup name
		engineGroup := new(group)
		engineGroup.engine = engine
		engineGroup.configGroup = configGroup

		// determine which lists belong to this group
		engineGroup.lists = assignedLists(configGroup.Lists, configGroup.SafeTags(), lists)

		// add created engine group to list of groups
		groups[idx] = engineGroup

		// set default group on engine if found
		if "default" == configGroup.Name {
			engine.defaultGroup = engineGroup
		}
	}

	// attach groups to consumers
	consumers := make([]*consumer, len(conf.Consumers))
	for index, configConsumer := range conf.Consumers {
		// create an active consumer
		consumer := new(consumer)
		consumer.engine = engine
		consumer.groupNames = make([]string, 0)
		consumer.resolverNames = make([]string, 0)
		consumer.configConsumer = configConsumer
		consumer.lists = make([]*config.GudgeonList, 0)

		// set as default consumer
		if strings.EqualFold(configConsumer.Name, "default") {
			engine.defaultConsumer = consumer
		}

		// link consumer to group when the consumer's group elements contains the group name
		for _, group := range groups {
			if util.StringIn(group.configGroup.Name, configConsumer.Groups) {
				consumer.groupNames = append(consumer.groupNames, group.configGroup.Name)

				// add resolvers from group too
				if len(group.configGroup.Resolvers) > 0 {
					consumer.resolverNames = append(consumer.resolverNames, group.configGroup.Resolvers...)
				}

				// add lists if they aren't already in the consumer lists
				for _, newList := range group.lists {
					listFound := false
					for _, currentList := range consumer.lists {
						if currentList.CanonicalName() == newList.CanonicalName() {
							listFound = true
							break
						}
					}
					if !listFound {
						consumer.lists = append(consumer.lists, newList)
					}
				}
			}
		}

		// add active consumer to list
		consumers[index] = consumer
	}

	// load lists (from remote urls)
	for _, list := range lists {
		// get list path
		path := conf.PathToList(list)

		// skip non-remote lists
		if !list.IsRemote() {
			continue
		}

		// skip downloading, don't need to download unless
		// certain conditions are met, which should be triggered
		// from inside the app or similar and not every time
		// an engine is created
		if _, err := os.Stat(path); err == nil {
			continue
		}

		// load/download list if required
		err := Download(engine, conf, list)
		if err != nil {
			return nil, err
		}
	}

	// create store based on gudgeon configuration and engine details
	// (requires lists to be downloaded and present before creation)
	engine.store = rule.CreateStoreWithMetrics(engine.Root(), conf, metrics)

	// set consumers as active on engine
	engine.consumers = consumers

	// force GC after loading the engine because
	// of all the extra allocation that gets performed
	// during the creation of the arrays and whatnot
	runtime.GC()

	return engine, nil
}
