package api

/************************************************
  generated by IDE. for [SIMAPI]
************************************************/

import (
	"github.com/sacloud/libsacloud/sacloud"
)

/************************************************
   To support fluent interface for Find()
************************************************/

// Reset 検索条件のリセット
func (api *SIMAPI) Reset() *SIMAPI {
	api.reset()
	return api
}

// Offset オフセット
func (api *SIMAPI) Offset(offset int) *SIMAPI {
	api.offset(offset)
	return api
}

// Limit リミット
func (api *SIMAPI) Limit(limit int) *SIMAPI {
	api.limit(limit)
	return api
}

// Include 取得する項目
func (api *SIMAPI) Include(key string) *SIMAPI {
	api.include(key)
	return api
}

// Exclude 除外する項目
func (api *SIMAPI) Exclude(key string) *SIMAPI {
	api.exclude(key)
	return api
}

// FilterBy 指定キーでのフィルター
func (api *SIMAPI) FilterBy(key string, value interface{}) *SIMAPI {
	api.filterBy(key, value, false)
	return api
}

// FilterMultiBy 任意項目でのフィルタ(完全一致 OR条件)
func (api *SIMAPI) FilterMultiBy(key string, value interface{}) *SIMAPI {
	api.filterBy(key, value, true)
	return api
}

// WithNameLike 名称条件
func (api *SIMAPI) WithNameLike(name string) *SIMAPI {
	return api.FilterBy("Name", name)
}

// WithTag タグ条件
func (api *SIMAPI) WithTag(tag string) *SIMAPI {
	return api.FilterBy("Tags.Name", tag)
}

// WithTags タグ(複数)条件
func (api *SIMAPI) WithTags(tags []string) *SIMAPI {
	return api.FilterBy("Tags.Name", []interface{}{tags})
}

// func (api *SIMAPI) WithSizeGib(size int) *SIMAPI {
// 	api.FilterBy("SizeMB", size*1024)
// 	return api
// }

// func (api *SIMAPI) WithSharedScope() *SIMAPI {
// 	api.FilterBy("Scope", "shared")
// 	return api
// }

// func (api *SIMAPI) WithUserScope() *SIMAPI {
// 	api.FilterBy("Scope", "user")
// 	return api
// }

// SortBy 指定キーでのソート
func (api *SIMAPI) SortBy(key string, reverse bool) *SIMAPI {
	api.sortBy(key, reverse)
	return api
}

// SortByName 名称でのソート
func (api *SIMAPI) SortByName(reverse bool) *SIMAPI {
	api.sortByName(reverse)
	return api
}

// func (api *SIMAPI) SortBySize(reverse bool) *SIMAPI {
// 	api.sortBy("SizeMB", reverse)
// 	return api
// }

/************************************************
   To support Setxxx interface for Find()
************************************************/

// SetEmpty 検索条件のリセット
func (api *SIMAPI) SetEmpty() {
	api.reset()
}

// SetOffset オフセット
func (api *SIMAPI) SetOffset(offset int) {
	api.offset(offset)
}

// SetLimit リミット
func (api *SIMAPI) SetLimit(limit int) {
	api.limit(limit)
}

// SetInclude 取得する項目
func (api *SIMAPI) SetInclude(key string) {
	api.include(key)
}

// SetExclude 除外する項目
func (api *SIMAPI) SetExclude(key string) {
	api.exclude(key)
}

// SetFilterBy 指定キーでのフィルター
func (api *SIMAPI) SetFilterBy(key string, value interface{}) {
	api.filterBy(key, value, false)
}

// SetFilterMultiBy 任意項目でのフィルタ(完全一致 OR条件)
func (api *SIMAPI) SetFilterMultiBy(key string, value interface{}) {
	api.filterBy(key, value, true)
}

// SetNameLike 名称条件
func (api *SIMAPI) SetNameLike(name string) {
	api.FilterBy("Name", name)
}

// SetTag タグ条件
func (api *SIMAPI) SetTag(tag string) {
	api.FilterBy("Tags.Name", tag)
}

// SetTags タグ(複数)条件
func (api *SIMAPI) SetTags(tags []string) {
	api.FilterBy("Tags.Name", []interface{}{tags})
}

// func (api *SIMAPI) SetSizeGib(size int)  {
// 	api.FilterBy("SizeMB", size*1024)
// }

// func (api *SIMAPI) SetSharedScope()  {
// 	api.FilterBy("Scope", "shared")
// }

// func (api *SIMAPI) SetUserScope()  {
// 	api.FilterBy("Scope", "user")
// }

// SetSortBy 指定キーでのソート
func (api *SIMAPI) SetSortBy(key string, reverse bool) {
	api.sortBy(key, reverse)
}

// SetSortByName 名称でのソート
func (api *SIMAPI) SetSortByName(reverse bool) {
	api.sortByName(reverse)
}

// func (api *SIMAPI) SetSortBySize(reverse bool)  {
// 	api.sortBy("SizeMB", reverse)
// }

/************************************************
  To support CRUD(Create/Read/Update/Delete)
************************************************/

// func (api *SIMAPI) New() *sacloud.SIM {
// 	return &sacloud.SIM{}
// }

// func (api *SIMAPI) Create(value *sacloud.SIM) (*sacloud.SIM, error) {
// 	return api.request(func(res *sacloud.Response) error {
// 		return api.create(api.createRequest(value), res)
// 	})
// }

// func (api *SIMAPI) Read(id string) (*sacloud.SIM, error) {
// 	return api.request(func(res *sacloud.Response) error {
// 		return api.read(id, nil, res)
// 	})
// }

// func (api *SIMAPI) Update(id string, value *sacloud.SIM) (*sacloud.SIM, error) {
// 	return api.request(func(res *sacloud.Response) error {
// 		return api.update(id, api.createRequest(value), res)
// 	})
// }

// func (api *SIMAPI) Delete(id string) (*sacloud.SIM, error) {
// 	return api.request(func(res *sacloud.Response) error {
// 		return api.delete(id, nil, res)
// 	})
// }

/************************************************
  Inner functions
************************************************/

func (api *SIMAPI) setStateValue(setFunc func(*sacloud.Request)) *SIMAPI {
	api.baseAPI.setStateValue(setFunc)
	return api
}

//func (api *SIMAPI) request(f func(*sacloud.Response) error) (*sacloud.SIM, error) {
//	res := &sacloud.Response{}
//	err := f(res)
//	if err != nil {
//		return nil, err
//	}
//	return res.SIM, nil
//}
//
//func (api *SIMAPI) createRequest(value *sacloud.SIM) *simRequest {
//	req := &simRequest{}
//	req.CommonServiceSIMItem = value
//	return req
//}
