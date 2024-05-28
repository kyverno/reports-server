package api

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

type subsetNegotiatedSerializer struct {
	accepts []func(info runtime.SerializerInfo) bool
	runtime.NegotiatedSerializer
}

func (s subsetNegotiatedSerializer) SupportedMediaTypes() []runtime.SerializerInfo {
	base := s.NegotiatedSerializer.SupportedMediaTypes()
	var filtered []runtime.SerializerInfo
	for _, info := range base {
		for _, accept := range s.accepts {
			if accept(info) {
				filtered = append(filtered, info)
				break
			}
		}
	}
	return filtered
}

func NoProtobuf(info runtime.SerializerInfo) bool {
	return info.MediaType != runtime.ContentTypeProtobuf
}

func SubsetNegotiatedSerializer(codecs serializer.CodecFactory, accepts ...func(info runtime.SerializerInfo) bool) runtime.NegotiatedSerializer {
	return subsetNegotiatedSerializer{accepts, codecs}
}

func DefaultSubsetNegotiatedSerializer(codecs serializer.CodecFactory) runtime.NegotiatedSerializer {
	return SubsetNegotiatedSerializer(codecs, NoProtobuf)
}
