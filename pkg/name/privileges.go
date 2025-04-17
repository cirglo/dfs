package name

type Privileges struct {
	Read   bool
	Write  bool
	Delete bool
}

func (p Privileges) Union(o Privileges) Privileges {
	return Privileges{
		Read:   p.Read && o.Read,
		Write:  p.Write && o.Write,
		Delete: p.Delete && o.Delete,
	}
}
