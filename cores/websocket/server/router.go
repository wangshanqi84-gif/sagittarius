package server

type Group struct {
	cores []core
	svr   *Engine
}

func (g *Group) Use(cores ...core) {
	g.cores = append(g.cores, cores...)
}

func (g *Group) WsGroup() *Group {
	group := &Group{
		svr:   g.svr,
		cores: nil,
	}
	if len(g.cores) > 0 {
		group.cores = append(group.cores, g.cores...)
	}
	return group
}

func (g *Group) Invoke(id int32, cores ...core) {
	var cs []core
	cs = append(cs, g.cores...)
	cs = append(cs, cores...)

	g.svr.addCore(id, cs...)
}

func (g *Group) Invokes(ids []int32, cores ...core) {
	var cs []core
	cs = append(cs, g.cores...)
	cs = append(cs, cores...)

	for _, id := range ids {
		g.svr.addCore(id, cs...)
	}
}

func (g *Group) Default(cores ...core) {
	g.svr.defaults = cores
}
