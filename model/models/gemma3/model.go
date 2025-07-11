package gemma3

import (
	"bytes"
	"image"
	"math"
	"slices"

	"github.com/YASSERRMD/Yolama/fs"
	"github.com/YASSERRMD/Yolama/kvcache"
	"github.com/YASSERRMD/Yolama/ml"
	"github.com/YASSERRMD/Yolama/ml/nn"
	"github.com/YASSERRMD/Yolama/model"
	"github.com/YASSERRMD/Yolama/model/input"
)

type Model struct {
	model.Base
	model.SentencePieceModel

	*VisionModel `gguf:"v,vision"`
	*TextModel

	*MultiModalProjector `gguf:"mm"`

	ImageProcessor
}

var _ model.MultimodalProcessor = (*Model)(nil)

type MultiModalProjector struct {
	SoftEmbNorm     *nn.RMSNorm `gguf:"mm_soft_emb_norm"`
	InputProjection *nn.Linear  `gguf:"mm_input_projection"`

	tokensPerImage int
}

func (p *MultiModalProjector) Forward(ctx ml.Context, visionOutputs ml.Tensor, imageSize, patchSize int, eps float32) ml.Tensor {
	l := visionOutputs.Dim(0)

	visionOutputs = visionOutputs.Permute(ctx, 1, 0, 2, 3).Contiguous(ctx)
	patchesPerImage := imageSize / patchSize
	visionOutputs = visionOutputs.Reshape(ctx, patchesPerImage, patchesPerImage, l)

	kernelSize := patchesPerImage / int(math.Sqrt(float64(p.tokensPerImage)))
	visionOutputs = visionOutputs.AvgPool2D(ctx, kernelSize, kernelSize, 0)
	visionOutputs = visionOutputs.Reshape(ctx, visionOutputs.Dim(0)*visionOutputs.Dim(1), l)
	visionOutputs = visionOutputs.Permute(ctx, 1, 0, 2, 3).Contiguous(ctx)
	visionOutputs = p.SoftEmbNorm.Forward(ctx, visionOutputs, eps)

	// TODO: inputProjection must be transposed since they're incompatible with visionOutputs
	visionOutputs = p.InputProjection.Weight.Permute(ctx, 1, 0, 2, 3).Contiguous(ctx).Mulmat(ctx, visionOutputs)
	return visionOutputs
}

func New(c fs.Config) (model.Model, error) {
	m := Model{
		SentencePieceModel: model.NewSentencePieceModel(
			&model.Vocabulary{
				Values: c.Strings("tokenizer.ggml.tokens"),
				Scores: c.Floats("tokenizer.ggml.scores"),
				Types:  c.Ints("tokenizer.ggml.token_type"),
				AddBOS: c.Bool("tokenizer.ggml.add_bos_token", true),
				BOS:    []int32{int32(c.Uint("tokenizer.ggml.bos_token_id"))},
				AddEOS: c.Bool("tokenizer.ggml.add_eos_token", false),
				EOS: append(
					[]int32{
						int32(c.Uint("tokenizer.ggml.eos_token_id")),
						int32(c.Uint("tokenizer.ggml.eot_token_id", 106)),
					},
					c.Ints("tokenizer.ggml.eos_token_ids")...,
				),
			},
		),
		ImageProcessor: newImageProcessor(c),
		VisionModel:    newVisionModel(c),
		TextModel:      newTextModel(c),
		MultiModalProjector: &MultiModalProjector{
			tokensPerImage: int(c.Uint("mm_tokens_per_image", 256)),
		},
	}

	slidingWindowLen := int32(c.Uint("attention.sliding_window"))
	m.Cache = kvcache.NewWrapperCache(kvcache.NewSWACache(slidingWindowLen, m.Shift), kvcache.NewCausalCache(m.Shift))

	return &m, nil
}

func (m *Model) EncodeMultimodal(ctx ml.Context, multimodalData []byte) ([]input.Multimodal, error) {
	if len(m.VisionModel.Layers) == 0 {
		return nil, model.ErrNoVisionModel
	}

	image, _, err := image.Decode(bytes.NewReader(multimodalData))
	if err != nil {
		return nil, err
	}

	f32s, err := m.ImageProcessor.ProcessImage(image)
	if err != nil {
		return nil, err
	}

	pixelValues := ctx.Input().FromFloatSlice(f32s,
		m.ImageProcessor.imageSize,
		m.ImageProcessor.imageSize,
		m.ImageProcessor.numChannels,
	)

	visionOutputs := m.VisionModel.Forward(ctx, pixelValues)
	visionOutputs = m.MultiModalProjector.Forward(ctx, visionOutputs, m.imageSize, m.patchSize, m.VisionModel.eps)
	return []input.Multimodal{{Tensor: visionOutputs}}, nil
}

func (m *Model) PostTokenize(inputs []input.Input) ([]input.Input, error) {
	var result []input.Input

	for _, inp := range inputs {
		if len(inp.Multimodal) == 0 {
			result = append(result, inp)
		} else {
			inputMultimodal := inp.Multimodal[0].Tensor

			result = append(result,
				input.Input{Token: 108, SameBatch: inputMultimodal.Dim(1) + 3}, // "\n\n"
				input.Input{Token: 255999},                                     // "<start_of_image>""
				input.Input{Multimodal: []input.Multimodal{{Tensor: inputMultimodal}}, MultimodalHash: inp.MultimodalHash}, // image data is on the first placeholder
			)

			// add image token placeholders
			result = append(result, slices.Repeat([]input.Input{{Token: 0}}, inputMultimodal.Dim(1)-1)...)

			result = append(result,
				input.Input{Token: 256000}, // <end_of_image>
				input.Input{Token: 108},    // "\n\n"
			)
		}
	}

	return result, nil
}

func (m *Model) Forward(ctx ml.Context, batch input.Batch) (ml.Tensor, error) {
	positions := ctx.Input().FromIntSlice(batch.Positions, len(batch.Positions))
	outputs := ctx.Input().FromIntSlice(batch.Outputs, len(batch.Outputs))

	return m.TextModel.Forward(ctx, batch.Inputs, positions, outputs, batch, m.Cache), nil
}

func init() {
	model.Register("gemma3", New)
}
