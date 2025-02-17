package cfg

func (mc *mConfig) loadFromText(parserf CfgParser, text ...string) (<-chan *CfgInfo, error) {
	sm, err := parserf(text...)
	if err != nil {
		return nil, err
	}
	chcfginfo := make(chan *CfgInfo)
	go func() {
		chcfginfo <- &CfgInfo{"": sm}
		close(chcfginfo)
	}()
	return chcfginfo, nil
}
