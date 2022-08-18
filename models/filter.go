package models

import (
	"fmt"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/filter"
)

// TagFilter is the name of a tag, and the values on which to filter
type TagFilter struct {
	filter filter.Filter
	Name   string
	Filter []string
}

// Filter containing drop/pass and tagdrop/tagpass rules
type Filter struct {
	fieldPass  filter.Filter
	nameDrop   filter.Filter
	tagInclude filter.Filter
	namePass   filter.Filter
	tagExclude filter.Filter
	fieldDrop  filter.Filter
	NameDrop   []string
	FieldPass  []string
	TagDrop    []TagFilter
	TagPass    []TagFilter
	TagExclude []string
	FieldDrop  []string
	TagInclude []string
	NamePass   []string
	isActive   bool
}

// Compile all Filter lists into filter.Filter objects.
func (f *Filter) Compile() error {
	if len(f.NameDrop) == 0 &&
		len(f.NamePass) == 0 &&
		len(f.FieldDrop) == 0 &&
		len(f.FieldPass) == 0 &&
		len(f.TagInclude) == 0 &&
		len(f.TagExclude) == 0 &&
		len(f.TagPass) == 0 &&
		len(f.TagDrop) == 0 {
		return nil
	}

	f.isActive = true
	var err error
	f.nameDrop, err = filter.Compile(f.NameDrop)
	if err != nil {
		return fmt.Errorf("error compiling 'namedrop': %w", err)
	}
	f.namePass, err = filter.Compile(f.NamePass)
	if err != nil {
		return fmt.Errorf("error compiling 'namepass': %w", err)
	}

	f.fieldDrop, err = filter.Compile(f.FieldDrop)
	if err != nil {
		return fmt.Errorf("error compiling 'fielddrop': %w", err)
	}
	f.fieldPass, err = filter.Compile(f.FieldPass)
	if err != nil {
		return fmt.Errorf("error compiling 'fieldpass': %w", err)
	}

	f.tagExclude, err = filter.Compile(f.TagExclude)
	if err != nil {
		return fmt.Errorf("error compiling 'tagexclude': %w", err)
	}
	f.tagInclude, err = filter.Compile(f.TagInclude)
	if err != nil {
		return fmt.Errorf("error compiling 'taginclude': %w", err)
	}

	for i := range f.TagDrop {
		f.TagDrop[i].filter, err = filter.Compile(f.TagDrop[i].Filter)
		if err != nil {
			return fmt.Errorf("error compiling 'tagdrop': %w", err)
		}
	}
	for i := range f.TagPass {
		f.TagPass[i].filter, err = filter.Compile(f.TagPass[i].Filter)
		if err != nil {
			return fmt.Errorf("error compiling 'tagpass': %w", err)
		}
	}
	return nil
}

// Select returns true if the metric matches according to the
// namepass/namedrop and tagpass/tagdrop filters.  The metric is not modified.
func (f *Filter) Select(metric cua.Metric) bool {
	if !f.isActive {
		return true
	}

	if !f.shouldNamePass(metric.Name()) {
		return false
	}

	if !f.shouldTagsPass(metric.TagList()) {
		return false
	}

	return true
}

// Modify removes any tags and fields from the metric according to the
// fieldpass/fielddrop and taginclude/tagexclude filters.
func (f *Filter) Modify(metric cua.Metric) {
	if !f.isActive {
		return
	}

	f.filterFields(metric)
	f.filterTags(metric)
}

// IsActive checking if filter is active
func (f *Filter) IsActive() bool {
	return f.isActive
}

// shouldNamePass returns true if the metric should pass, false if should drop
// based on the drop/pass filter parameters
func (f *Filter) shouldNamePass(key string) bool {
	pass := func(f *Filter) bool {
		return f.namePass.Match(key)
	}

	drop := func(f *Filter) bool {
		return f.nameDrop.Match(key)
	}

	switch {
	case f.namePass != nil && f.nameDrop != nil:
		return pass(f) && drop(f)
	case f.namePass != nil:
		return pass(f)
	case f.nameDrop != nil:
		return drop(f)
	}

	return true
}

// shouldFieldPass returns true if the metric should pass, false if should drop
// based on the drop/pass filter parameters
func (f *Filter) shouldFieldPass(key string) bool {
	switch {
	case f.fieldPass != nil && f.fieldDrop != nil:
		return f.fieldPass.Match(key) && !f.fieldDrop.Match(key)
	case f.fieldPass != nil:
		return f.fieldPass.Match(key)
	case f.fieldDrop != nil:
		return !f.fieldDrop.Match(key)
	}
	return true
}

// shouldTagsPass returns true if the metric should pass, false if should drop
// based on the tagdrop/tagpass filter parameters
func (f *Filter) shouldTagsPass(tags []*cua.Tag) bool {
	pass := func(f *Filter) bool {
		for _, pat := range f.TagPass {
			if pat.filter == nil {
				continue
			}
			for _, tag := range tags {
				if tag.Key == pat.Name {
					if pat.filter.Match(tag.Value) {
						return true
					}
				}
			}
		}
		return false
	}

	drop := func(f *Filter) bool {
		for _, pat := range f.TagDrop {
			if pat.filter == nil {
				continue
			}
			for _, tag := range tags {
				if tag.Key == pat.Name {
					if pat.filter.Match(tag.Value) {
						return false
					}
				}
			}
		}
		return true
	}

	// Add additional logic in case where both parameters are set.
	switch {
	case f.TagPass != nil && f.TagDrop != nil:
		// return true only in case when tag pass and won't be dropped (true, true).
		// in case when the same tag should be passed and dropped it will be dropped (true, false).
		return pass(f) && drop(f)
	case f.TagPass != nil:
		return pass(f)
	case f.TagDrop != nil:
		return drop(f)
	}

	return true
}

// filterFields removes fields according to fieldpass/fielddrop.
func (f *Filter) filterFields(metric cua.Metric) {
	filterKeys := []string{}
	for _, field := range metric.FieldList() {
		if !f.shouldFieldPass(field.Key) {
			filterKeys = append(filterKeys, field.Key)
		}
	}

	for _, key := range filterKeys {
		metric.RemoveField(key)
	}
}

// filterTags removes tags according to taginclude/tagexclude.
func (f *Filter) filterTags(metric cua.Metric) {
	filterKeys := []string{}
	if f.tagInclude != nil {
		for _, tag := range metric.TagList() {
			if !f.tagInclude.Match(tag.Key) {
				filterKeys = append(filterKeys, tag.Key)
			}
		}
	}
	for _, key := range filterKeys {
		metric.RemoveTag(key)
	}

	if f.tagExclude != nil {
		for _, tag := range metric.TagList() {
			if f.tagExclude.Match(tag.Key) {
				filterKeys = append(filterKeys, tag.Key)
			}
		}
	}
	for _, key := range filterKeys {
		metric.RemoveTag(key)
	}
}
