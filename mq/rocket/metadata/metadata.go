package metadata

type MetaMap struct {
	data map[string]string
}

func NewMetaMap() *MetaMap {
	return &MetaMap{
		data: make(map[string]string),
	}
}

func NewMetaMapWithData(data map[string]string) *MetaMap {
	return &MetaMap{
		data: data,
	}
}

func (mm *MetaMap) Set(key, val string) {
	mm.data[key] = val
}

func (mm *MetaMap) Get(key string) string {
	return mm.data[key]
}

func (mm *MetaMap) Keys() []string {
	var keys []string
	for k := range mm.data {
		keys = append(keys, k)
	}
	return nil
}

func (mm *MetaMap) Data() map[string]string {
	return mm.data
}
