//go:build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ParcaScrapeConfig) DeepCopyInto(out *ParcaScrapeConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ParcaScrapeConfig.
func (in *ParcaScrapeConfig) DeepCopy() *ParcaScrapeConfig {
	if in == nil {
		return nil
	}
	out := new(ParcaScrapeConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ParcaScrapeConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ParcaScrapeConfigList) DeepCopyInto(out *ParcaScrapeConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ParcaScrapeConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ParcaScrapeConfigList.
func (in *ParcaScrapeConfigList) DeepCopy() *ParcaScrapeConfigList {
	if in == nil {
		return nil
	}
	out := new(ParcaScrapeConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ParcaScrapeConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ParcaScrapeConfigSpec) DeepCopyInto(out *ParcaScrapeConfigSpec) {
	*out = *in
	in.Selector.DeepCopyInto(&out.Selector)
	in.ScrapeConfig.DeepCopyInto(&out.ScrapeConfig)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ParcaScrapeConfigSpec.
func (in *ParcaScrapeConfigSpec) DeepCopy() *ParcaScrapeConfigSpec {
	if in == nil {
		return nil
	}
	out := new(ParcaScrapeConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ParcaScrapeConfigStatus) DeepCopyInto(out *ParcaScrapeConfigStatus) {
	*out = *in
	if in.PodIPs != nil {
		in, out := &in.PodIPs, &out.PodIPs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ParcaScrapeConfigStatus.
func (in *ParcaScrapeConfigStatus) DeepCopy() *ParcaScrapeConfigStatus {
	if in == nil {
		return nil
	}
	out := new(ParcaScrapeConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in PprofConfig) DeepCopyInto(out *PprofConfig) {
	{
		in := &in
		*out = make(PprofConfig, len(*in))
		for key, val := range *in {
			var outVal *PprofProfilingConfig
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = new(PprofProfilingConfig)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PprofConfig.
func (in PprofConfig) DeepCopy() PprofConfig {
	if in == nil {
		return nil
	}
	out := new(PprofConfig)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PprofProfilingConfig) DeepCopyInto(out *PprofProfilingConfig) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PprofProfilingConfig.
func (in *PprofProfilingConfig) DeepCopy() *PprofProfilingConfig {
	if in == nil {
		return nil
	}
	out := new(PprofProfilingConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProfilingConfig) DeepCopyInto(out *ProfilingConfig) {
	*out = *in
	if in.PprofConfig != nil {
		in, out := &in.PprofConfig, &out.PprofConfig
		*out = make(PprofConfig, len(*in))
		for key, val := range *in {
			var outVal *PprofProfilingConfig
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = new(PprofProfilingConfig)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProfilingConfig.
func (in *ProfilingConfig) DeepCopy() *ProfilingConfig {
	if in == nil {
		return nil
	}
	out := new(ProfilingConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RelabelConfig) DeepCopyInto(out *RelabelConfig) {
	*out = *in
	if in.SourceLabels != nil {
		in, out := &in.SourceLabels, &out.SourceLabels
		*out = make([]LabelName, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RelabelConfig.
func (in *RelabelConfig) DeepCopy() *RelabelConfig {
	if in == nil {
		return nil
	}
	out := new(RelabelConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScrapeConfig) DeepCopyInto(out *ScrapeConfig) {
	*out = *in
	if in.Params != nil {
		in, out := &in.Params, &out.Params
		*out = make(map[string][]string, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
	if in.ProfilingConfig != nil {
		in, out := &in.ProfilingConfig, &out.ProfilingConfig
		*out = new(ProfilingConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.RelabelConfigs != nil {
		in, out := &in.RelabelConfigs, &out.RelabelConfigs
		*out = make([]*RelabelConfig, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(RelabelConfig)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	if in.StaticConfigs != nil {
		in, out := &in.StaticConfigs, &out.StaticConfigs
		*out = make([]StaticConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScrapeConfig.
func (in *ScrapeConfig) DeepCopy() *ScrapeConfig {
	if in == nil {
		return nil
	}
	out := new(ScrapeConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StaticConfig) DeepCopyInto(out *StaticConfig) {
	*out = *in
	if in.Targets != nil {
		in, out := &in.Targets, &out.Targets
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StaticConfig.
func (in *StaticConfig) DeepCopy() *StaticConfig {
	if in == nil {
		return nil
	}
	out := new(StaticConfig)
	in.DeepCopyInto(out)
	return out
}
