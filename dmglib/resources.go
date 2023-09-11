package dmglib

type ResourceData struct {
	attributes string
	cfname     string
	data       string
	id         string
	name       string
}

type Resource struct {
	data []ResourceData
}

type Resources struct {
	resources []Resource
}

func parseResources(unstructured map[string]interface{}) (*Resources, error) {
	res := new(Resources)
	return res, nil
}

func (r *Resources) getResourceData(name string) (ResourceData, error) {
	// TODO: implement me properly
	// return r.resources[0].data[0], nil
	return ResourceData{attributes: "", cfname: "", data: "", id: "", name: ""}, nil
}
