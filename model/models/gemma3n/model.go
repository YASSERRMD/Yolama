package gemma3n

import (
	"github.com/YASSERRMD/Yolama/fs"
	"github.com/YASSERRMD/Yolama/kvcache"
	"github.com/YASSERRMD/Yolama/ml"
	"github.com/YASSERRMD/Yolama/model"
	"github.com/YASSERRMD/Yolama/model/input"
)

type Model struct {
	model.Base
	model.SentencePieceModel

	*TextModel
}

// Forward implements model.Model.
func (m *Model) Forward(ctx ml.Context, batch input.Batch) (ml.Tensor, error) {
	return m.TextModel.Forward(ctx, batch, m.Cache)
}

func New(c fs.Config) (model.Model, error) {
	m := Model{
		TextModel: newTextModel(c),
		SentencePieceModel: model.NewSentencePieceModel(
			&model.Vocabulary{
				Values: c.Strings("tokenizer.ggml.tokens"),
				Scores: c.Floats("tokenizer.ggml.scores"),
				Types:  c.Ints("tokenizer.ggml.token_type"),
				AddBOS: c.Bool("tokenizer.ggml.add_bos_token", true),
				BOS:    []int32{int32(c.Uint("tokenizer.ggml.bos_token_id"))},
				AddEOS: c.Bool("tokenizer.ggml.add_eos_token", false),
				EOS: append(
					[]int32{int32(c.Uint("tokenizer.ggml.eos_token_id"))},
					c.Ints("tokenizer.ggml.eos_token_ids")...,
				),
			},
		),
	}

	m.Cache = kvcache.NewWrapperCache(
		kvcache.NewCausalCache(m.Shift),
		kvcache.NewSWACache(int32(c.Uint("attention.sliding_window")), m.Shift),
	)
	return &m, nil
}

func init() {
	model.Register("gemma3n", New)
}
