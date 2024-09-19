package cmn

type NetworkFoundationOption func(nf *NetworkFoundation) error

func WithRobotCfg(robotCfg *Robot) NetworkFoundationOption {
	return func(nf *NetworkFoundation) error {
		nf.Robot = robotCfg
		return nil
	}
}

func WithChannelTransferCfg(ctCfg *ChannelTransfer) NetworkFoundationOption {
	return func(nf *NetworkFoundation) error {
		nf.ChannelTransfer = ctCfg
		return nil
	}
}

func WithRobotTemplate(robotTemplate string) NetworkFoundationOption {
	return func(nf *NetworkFoundation) error {
		if robotTemplate != "" {
			nf.Templates.Robot = robotTemplate
		}
		return nil
	}
}

func WithChannelTransferTemplate(ctTemplate string) NetworkFoundationOption {
	return func(nf *NetworkFoundation) error {
		if ctTemplate != "" {
			nf.Templates.ChannelTransfer = ctTemplate
		}
		return nil
	}
}
