package dpos

type API struct {
	dpos *Dpos
}


func (api *API) Test() error {
	api.dpos.Close()
	return nil
}
