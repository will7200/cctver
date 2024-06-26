package pipeline

import (
	"context"

	"github.com/go-gst/go-glib/glib"
	"github.com/go-gst/go-gst/gst"
	"github.com/rs/zerolog/log"
)

// Link struct holds elements that should
// be linked left -> right
type Link struct {
	left  *Element
	right *Element
}

type LinkWithCaps struct {
	left   *Element
	right  *Element
	filter *gst.Caps
}

// Pipeline wrapper around go-gst
type Pipeline struct {
	name     string
	elements []*Element
	pipeline *gst.Pipeline
	watches  []func(message *gst.Message) bool
	// partials hold partial pipelines
	partials []PartialPipeline

	// have we called built
	built bool
	// loop
	loop *glib.MainLoop
}

// Build the pipeline, once built it will never be built again
func (p *Pipeline) Build() error {
	var err error
	if p.built == true {
		return nil
	}
	for _, partial := range p.partials {
		if err = partial.Prepare(p); err != nil {
			return err
		}
	}
	p.pipeline, err = gst.NewPipeline(p.name)
	if err != nil {
		return err
	}
	for _, element := range p.elements {
		err = element.Build()
		if err != nil {
			return err
		}
	}
	for index := range p.elements {
		err := p.pipeline.Add(p.elements[index].el)
		if err != nil {
			return err
		}
	}
	for _, partial := range p.partials {
		if err = partial.Build(p); err != nil {
			return err
		}
	}
	p.built = true
	return nil
}

// AddWatch adds watch function for the event bus
func (p *Pipeline) AddWatch(watch func(element *gst.Message) bool) {
	p.watches = append(p.watches, watch)
}

// AddElements to the pipeline
func (p *Pipeline) AddElements(elements ...*Element) {
	p.elements = append(p.elements, elements...)
}

// AddPartialPipeline to the pipeline
func (p *Pipeline) AddPartialPipeline(partial PartialPipeline) {
	p.partials = append(p.partials, partial)
}

// Start the pipeline
func (p *Pipeline) Start(ctx context.Context, mainLoop *glib.MainLoop) (rerr error) {
	var err error
	err = p.Build()
	if err != nil {
		return err
	}
	p.loop = mainLoop
	pipeline := p.pipeline
	logger := log.With().Str("pipeline", p.name).Logger()
	// Add a message handler to the pipeline bus, logging interesting information to the console.
	pipeline.GetPipelineBus().AddWatch(func(msg *gst.Message) bool {
		switch msg.Type() {
		case gst.MessageEOS: // When end-of-stream is received stop the main loop
			logger.Print("End of Stream")
			pipeline.BlockSetState(gst.StateNull)
			mainLoop.Quit()
		case gst.MessageError: // Error messages are always fatal
			err := msg.ParseError()
			logger.Err(err).Msg("error from gst")
			if debug := err.DebugString(); debug != "" {
				logger.Debug().Msg(debug)
			}
			rerr = err
			mainLoop.Quit()
		default:
			// All messages implement a Stringer. However, this is
			// typically an expensive thing to do and should be avoided.
			logger.Println(msg.String())

		}
		for _, watch := range p.watches {
			watch(msg)
		}
		return true
	})
	for _, partial := range p.partials {
		if err = partial.Start(ctx, p); err != nil {
			return err
		}
	}
	err = pipeline.SetState(gst.StateReady)
	if err != nil {
		return err
	}
	logger.Print("Starting pipeline")
	// Start the pipeline
	err = pipeline.SetState(gst.StatePlaying)
	if err != nil {
		return err
	}

	// Block on the main loop
	mainLoop.RunError()
	return rerr
}

func (p *Pipeline) Finish(ctx context.Context) {
	for _, partial := range p.partials {
		if err := partial.Stop(ctx, p); err != nil {
			log.Err(err).Msg("unable to stop partial pipeline process")
		}
	}
}

// Quit the current pipeline
func (p *Pipeline) Quit() {
	if p.loop == nil {
		panic("Unable to quit")
	}
	p.loop.Quit()
}

// NewPipeline returns a new pipeline given a name
func NewPipeline(name string) *Pipeline {
	return &Pipeline{
		name:  name,
		built: false,
	}
}
