package console

import "time"

type MetricsService struct {
	client *Client
}

type Instance struct {
	CreatedAt    time.Time `json:"createdAt"`
	GUID         string    `json:"guid"`
	Name         string    `json:"name"`
	Organization string    `json:"organization"`
	Space        string    `json:"space"`
}

type MetricsResponse struct {
	Data struct {
		Instances []Instance `json:"instances"`
	} `json:"data"`
	Status string `json:"status"`
}

type Group struct {
	Name  string `json:"name"`
	Rules []Rule `json:"rules"`
}

type RuleResponse struct {
	Data struct {
		Groups []Group `json:"groups"`
	} `json:"data"`
	Status string `json:"status"`
}

type Rule struct {
	Annotations struct {
		Description string `json:"description"`
		Resolved    string `json:"resolved"`
		Summary     string `json:"summary"`
	} `json:"annotations"`
	Description string `json:"description"`
	ID          string `json:"id"`
	Metric      string `json:"metric"`
	Rule        struct {
		ExtraFor []struct {
			Name         string   `json:"name"`
			Options      []string `json:"options"`
			Type         string   `json:"type"`
			VariableName string   `json:"variableName"`
		} `json:"extraFor"`
		Extras []struct {
			Name         string   `json:"name"`
			Options      []string `json:"options"`
			Type         string   `json:"type"`
			VariableName string   `json:"variableName"`
		} `json:"extras"`
		Operators []string `json:"operators"`
		Subject   string   `json:"subject"`
		Threshold struct {
			Default int      `json:"default"`
			Max     int      `json:"max"`
			Min     int      `json:"min"`
			Type    string   `json:"type"`
			Unit    []string `json:"unit"`
		} `json:"threshold"`
	} `json:"rule"`
	Template string `json:"template"`
}

// GetGroupedRules looks up available rules
func (c *MetricsService) GetGroupedRules(options ...OptionFunc) (*[]Group, *Response, error) {
	req, err := c.client.NewRequest(CONSOLE, "GET", "v3/metrics/rules", nil, options)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	var response RuleResponse

	resp, err := c.client.Do(req, &response)
	if err != nil {
		return nil, resp, err
	}
	return &response.Data.Groups, resp, err
}

// GetRuleByID looks up available instances
func (c *MetricsService) GetRuleByID(id string, options ...OptionFunc) (*Rule, *Response, error) {
	req, err := c.client.NewRequest(CONSOLE, "GET", "v3/metrics/rules/"+id, nil, options)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	var response struct {
		Data   Rule   `json:"data"`
		Status string `json:"status"`
	}

	resp, err := c.client.Do(req, &response)
	if err != nil {
		return nil, resp, err
	}
	return &response.Data, resp, err
}

// GetInstances looks up available instances
func (c *MetricsService) GetInstances(options ...OptionFunc) (*[]Instance, *Response, error) {
	req, err := c.client.NewRequest(CONSOLE, "GET", "v3/metrics/instances", nil, options)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	var response MetricsResponse

	resp, err := c.client.Do(req, &response)
	if err != nil {
		return nil, resp, err
	}
	return &response.Data.Instances, resp, err
}

// GetInstanceByID looks up an instance by ID
func (c *MetricsService) GetInstanceByID(id string, options ...OptionFunc) (*Instance, *Response, error) {
	req, err := c.client.NewRequest(CONSOLE, "GET", "v3/metrics/intances/"+id, nil, options)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	var response struct {
		Data   Instance `json:"data"`
		Status string   `json:"status"`
	}

	resp, err := c.client.Do(req, &response)
	if err != nil {
		return nil, resp, err
	}
	return &response.Data, resp, err
}
