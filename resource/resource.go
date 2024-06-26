package resource

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"time"

	"github.com/rs/rest-layer/schema"
	"github.com/rs/rest-layer/schema/query"
)

// Resource holds information about a class of items exposed on the API.
type Resource struct {
	parentField string
	name        string
	path        string
	schema      schema.Schema
	validator   validatorFallback
	storage     storageHandler
	conf        Conf
	resources   subResources
	aliases     map[string]url.Values
	hooks       eventHandler
	middlewares middlewareHandlers
	commands    map[string]Command
}

type Command func(ctx context.Context, r *http.Request, item *Item, payload map[string]interface{}) (http.Header, *Item, map[string]interface{}, error)
type subResources []*Resource

// get gets a sub resource by its name.
func (sr subResources) get(name string) *Resource {
	i := sort.Search(len(sr), func(i int) bool {
		return sr[i].name >= name
	})
	if i >= len(sr) {
		return nil
	}
	r := sr[i]
	if r.name != name {
		return nil
	}
	return r
}

// add adds the resource to the subResources in a pre-sorted way.
func (sr *subResources) add(rsrc *Resource) {
	for i, r := range *sr {
		if rsrc.name < r.name {
			*sr = append((*sr)[:i], append(subResources{rsrc}, (*sr)[i:]...)...)
			return
		}
	}
	*sr = append(*sr, rsrc)
}

// validatorFallback wraps a validator and fallback on given schema if the GetField
// returns nil on a given name.
type validatorFallback struct {
	schema.Validator
	fallback schema.Schema
}

func (v validatorFallback) GetField(name string) *schema.Field {
	if f := v.Validator.GetField(name); f != nil {
		return f
	}
	return v.fallback.GetField(name)
}

// newResource creates a new resource with provided spec, handler and config.
func newResource(name string, s schema.Schema, h Storer, c Conf) *Resource {
	r := &Resource{
		name:   name,
		path:   name,
		schema: s,
		validator: validatorFallback{
			Validator: s,
			fallback:  schema.Schema{Fields: schema.Fields{}},
		},
		storage:   storageWrapper{h},
		conf:      c,
		resources: subResources{},
		aliases:   map[string]url.Values{},
		commands:  map[string]Command{},
	}
	initMiddlewares(r)
	return r
}

// Name returns the name of the resource
func (r *Resource) Name() string {
	return r.name
}

// Path returns the full path of the resource composed of names of each
// intermediate resources separated by dots (i.e.: res1.res2.res3).
func (r *Resource) Path() string {
	return r.path
}

// ParentField returns the name of the field on which the resource is bound to
// its parent if any.
func (r *Resource) ParentField() string {
	return r.parentField
}

// Compile the resource graph and report any error.
func (r *Resource) Compile(rc schema.ReferenceChecker) error {
	// Compile schema and panic on any compilation error.
	if c, ok := r.validator.Validator.(schema.Compiler); ok {
		if err := c.Compile(rc); err != nil {
			return fmt.Errorf(": schema compilation error: %s", err)
		}
	}
	for _, r := range r.resources {
		if err := r.Compile(rc); err != nil {
			if err.Error()[0] == ':' {
				// Check if I'm the direct ancestor of the raised sub-error.
				return fmt.Errorf("%s%s", r.name, err)
			}
			return fmt.Errorf("%s.%s", r.name, err)
		}
	}
	return nil
}

// Bind a sub-resource with the provided name. The field parameter defines the parent
// resource's which contains the sub resource id.
//
//	users := api.Bind("users", userSchema, userHandler, userConf)
//	// Bind a sub resource on /users/:user_id/posts[/:post_id]
//	// and reference the user on each post using the "user" field.
//	posts := users.Bind("posts", "user", postSchema, postHandler, postConf)
//
// This method will panic an alias or a resource with the same name is already bound
// or if the specified field doesn't exist in the parent resource spec.
func (r *Resource) Bind(name, field string, s schema.Schema, h Storer, c Conf) *Resource {
	assertNotBound(name, r.resources, r.aliases)
	if f := s.GetField(field); f == nil {
		logPanicf(nil, "Cannot bind `%s' as sub-resource: field `%s' does not exist in the sub-resource'", name, field)
	}
	sr := newResource(name, s, h, c)
	sr.parentField = field
	sr.path = r.path + "." + name
	r.resources.add(sr)
	r.validator.fallback.Fields[name] = schema.Field{
		ReadOnly: true,
		Validator: &schema.Connection{
			Path:      "." + name,
			Field:     field,
			Validator: sr.validator,
		},
		Params: schema.Params{
			"skip": schema.Param{
				Description: "The number of items to skip",
				Validator: schema.Integer{
					Boundaries: &schema.Boundaries{Min: 0},
				},
			},
			"page": schema.Param{
				Description: "The page number",
				Validator: schema.Integer{
					Boundaries: &schema.Boundaries{Min: 1, Max: 1000},
				},
			},
			"limit": schema.Param{
				Description: "The number of items to return per page",
				Validator: schema.Integer{
					Boundaries: &schema.Boundaries{Min: 0, Max: 1000},
				},
			},
			"sort": schema.Param{
				Description: "The field(s) to sort on",
				Validator:   schema.String{},
			},
			"filter": schema.Param{
				Description: "The filter query",
				Validator:   schema.String{},
			},
		},
	}
	return sr
}

// GetResources returns first level resources.
func (r *Resource) GetResources() []*Resource {
	return r.resources
}

// Alias adds an pre-built resource query on /<resource>/<alias>.
//
//	// Add a friendly alias to public posts on /users/:user_id/posts/public
//	// (equivalent to /users/:user_id/posts?filter={"public":true})
//	posts.Alias("public", url.Values{"where": []string{"{\"public\":true}"}})
//
// This method will panic an alias or a resource with the same name is already bound.
func (r *Resource) Alias(name string, v url.Values) {
	assertNotBound(name, r.resources, r.aliases)
	r.aliases[name] = v
}

// GetAlias returns the alias set for the name if any.
func (r *Resource) GetAlias(name string) (url.Values, bool) {
	a, found := r.aliases[name]
	return a, found
}

// GetAliases returns all the alias names set on the resource.
func (r *Resource) GetAliases() []string {
	n := make([]string, 0, len(r.aliases))
	for a := range r.aliases {
		n = append(n, a)
	}
	return n
}

func (r *Resource) Command(name string, command Command) {
	// allow only A-Z,a-z,0-9, _ and - in command name. Panic if not
	matched, _ := regexp.MatchString(`^[A-Za-z0-9_\-]+$`, name)
	if !matched {
		logPanicf(context.Background(), "Invalid command name: %s for resource %s", name, r.name)
	}

	assertNotBound(name, r.resources, r.aliases)
	r.commands[name] = command
}

func (r *Resource) GetCommand(name string) (Command, bool) {
	c, found := r.commands[name]
	return c, found
}

func (r *Resource) GetCommands() []Command {
	n := make([]Command, 0, len(r.commands))
	for _, c := range r.commands {
		n = append(n, c)
	}
	return n
}

// Schema returns the resource's schema.
func (r *Resource) Schema() schema.Schema {
	return r.schema
}

// Validator returns the resource's validator.
func (r *Resource) Validator() schema.Validator {
	return r.validator
}

// Conf returns the resource's configuration.
func (r *Resource) Conf() Conf {
	return r.conf
}

// Use attaches an event handler to the resource. This event handler must
// implement on of the resource.*EventHandler interface or this method returns
// an error.
func (r *Resource) Use(e interface{}) error {
	return r.hooks.use(e)
}

// Get get one item by its id. If item is not found, ErrNotFound error is
// returned.
func (r *Resource) Get(ctx context.Context, id interface{}) (item *Item, err error) {
	if LoggerLevel <= LogLevelDebug && Logger != nil {
		defer func(t time.Time) {
			Logger(ctx, LogLevelDebug, fmt.Sprintf("%s.Get(%v)", r.path, id), map[string]interface{}{
				"duration": time.Since(t),
				"error":    err,
			})
		}(time.Now())
	}
	item, err = r.middlewares.onGetThen(ctx, id)
	return
}

// MultiGet get some items by their id and return them in the same order. If one
// or more item(s) is not found, their slot in the response is set to nil.
func (r *Resource) MultiGet(ctx context.Context, ids []interface{}) (items []*Item, err error) {
	if LoggerLevel <= LogLevelDebug && Logger != nil {
		defer func(t time.Time) {
			Logger(ctx, LogLevelDebug, fmt.Sprintf("%s.MultiGet(%v)", r.path, ids), map[string]interface{}{
				"duration": time.Since(t),
				"found":    len(items),
				"error":    err,
			})
		}(time.Now())
	}
	items, err = r.middlewares.onMultiGetThen(ctx, ids)
	return
}

// Find calls the Find method on the storage handler with the corresponding pre/post hooks.
func (r *Resource) Find(ctx context.Context, q *query.Query) (list *ItemList, err error) {
	return r.find(ctx, q, false)
}

// FindWithTotal calls the Find method on the storage handler with the
// corresponding pre/post hooks. If the storage is not able to compute the
// total, this method will call the Count method on the storage. If the storage
// Find does not compute the total and the Counter interface is not implemented,
// an ErrNotImplemented error is returned.
func (r *Resource) FindWithTotal(ctx context.Context, q *query.Query) (list *ItemList, err error) {
	return r.find(ctx, q, true)
}

func (r *Resource) find(ctx context.Context, q *query.Query, forceTotal bool) (list *ItemList, err error) {
	if LoggerLevel <= LogLevelDebug && Logger != nil {
		defer func(t time.Time) {
			found := -1
			if list != nil {
				found = len(list.Items)
			}
			Logger(ctx, LogLevelDebug, fmt.Sprintf("%s.Find(...)", r.path), map[string]interface{}{
				"duration": time.Since(t),
				"found":    found,
				"error":    err,
			})
		}(time.Now())
	}
	list, err = r.middlewares.onFindThen(ctx, q, forceTotal)
	return
}

type ReducerFunc = func(item *Item) error

// Reduce calls the Reduce method on the storage handler with the corresponding without hooks.
// Reduce does not return `Total` number of items. You need to call `Count` method to get it.
func (r *Resource) Reduce(ctx context.Context, q *query.Query, reducer ReducerFunc) (err error) {
	if LoggerLevel <= LogLevelDebug && Logger != nil {
		defer func(t time.Time) {
			Logger(ctx, LogLevelDebug, fmt.Sprintf("%s.Reduce(...)", r.path), map[string]interface{}{
				"duration": time.Since(t),
				"error":    err,
			})
		}(time.Now())
	}
	err = r.middlewares.onReduceThen(ctx, q, reducer)
	return

}

// Insert implements Storer interface.
func (r *Resource) Insert(ctx context.Context, items []*Item) (err error) {
	if LoggerLevel <= LogLevelDebug && Logger != nil {
		defer func(t time.Time) {
			Logger(ctx, LogLevelDebug, fmt.Sprintf("%s.Insert(items[%d])", r.path, len(items)), map[string]interface{}{
				"duration": time.Since(t),
				"error":    err,
			})
		}(time.Now())
	}
	var newItems []*Item
	newItems, err = r.middlewares.onInsertThen(ctx, items)
	if err == nil {
		copy(items, newItems)
	}
	return
}

func recalcEtag(items []*Item) error {
	if items == nil {
		return nil
	}

	for _, v := range items {
		if v == nil {
			continue
		}
		etag, err := GenEtag(v.Payload)
		if err != nil {
			return err
		}
		v.ETag = etag
	}
	return nil
}

// Update implements Storer interface.
func (r *Resource) Update(ctx context.Context, item *Item, original *Item) (err error) {
	if LoggerLevel <= LogLevelDebug && Logger != nil {
		defer func(t time.Time) {
			Logger(ctx, LogLevelDebug, fmt.Sprintf("%s.Update(%v, %v)", r.path, item.ID, original.ID), map[string]interface{}{
				"duration": time.Since(t),
				"error":    err,
			})
		}(time.Now())
	}
	var newItem *Item
	newItem, err = r.middlewares.onUpdateThen(ctx, item, original)
	if err == nil {
		*item = *newItem
	}
	return
}

// Delete implements Storer interface.
func (r *Resource) Delete(ctx context.Context, item *Item) (err error) {
	if LoggerLevel <= LogLevelDebug && Logger != nil {
		defer func(t time.Time) {
			Logger(ctx, LogLevelDebug, fmt.Sprintf("%s.Delete(%v)", r.path, item.ID), map[string]interface{}{
				"duration": time.Since(t),
				"error":    err,
			})
		}(time.Now())
	}
	var newItem *Item
	newItem, err = r.middlewares.onDeleteThen(ctx, item)
	if err == nil {
		*item = *newItem
	}
	return
}

// Clear implements Storer interface.
func (r *Resource) Clear(ctx context.Context, q *query.Query) (deleted int, err error) {
	if LoggerLevel <= LogLevelDebug && Logger != nil {
		defer func(t time.Time) {
			Logger(ctx, LogLevelDebug, fmt.Sprintf("%s.Clear(%v)", r.path, q), map[string]interface{}{
				"duration": time.Since(t),
				"deleted":  deleted,
				"error":    err,
			})
		}(time.Now())
	}
	deleted, err = r.middlewares.onClearThen(ctx, q)
	return
}

// Count implements Counter interface.
func (r *Resource) Count(ctx context.Context, q *query.Query) (total int, err error) {
	if LoggerLevel <= LogLevelDebug && Logger != nil {
		defer func(t time.Time) {
			Logger(ctx, LogLevelDebug, fmt.Sprintf("%s.Clear(%v)", r.path, q), map[string]interface{}{
				"duration": time.Since(t),
				"total":    total,
				"error":    err,
			})
		}(time.Now())
	}
	total, err = r.storage.Count(ctx, q)
	return
}
