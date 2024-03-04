package resource

import (
	"context"
	"fmt"

	"github.com/rs/rest-layer/schema/query"
)

type OnGetMiddlewareHandler func(ctx context.Context, id interface{}) (*Item, error)
type OnGetMiddleware func(next OnGetMiddlewareHandler) OnGetMiddlewareHandler

type OnMultiGetMiddlewareHandler func(ctx context.Context, ids []interface{}) ([]*Item, error)
type OnMultiGetMiddleware func(next OnMultiGetMiddlewareHandler) OnMultiGetMiddlewareHandler

type OnFindMiddlewareHandler func(ctx context.Context, q *query.Query, forceTotal bool) (*ItemList, error)
type OnFindMiddleware func(next OnFindMiddlewareHandler) OnFindMiddlewareHandler

type OnReduceMiddlewareHandler func(ctx context.Context, q *query.Query, reducer ReducerFunc) error
type OnReduceMiddleware func(next OnReduceMiddlewareHandler) OnReduceMiddlewareHandler

type OnInsertMiddlewareHandler func(ctx context.Context, items []*Item) ([]*Item, error)
type OnInsertMiddleware func(next OnInsertMiddlewareHandler) OnInsertMiddlewareHandler

type OnUpdateMiddlewareHandler func(ctx context.Context, item *Item, original *Item) (*Item, error)
type OnUpdateMiddleware func(next OnUpdateMiddlewareHandler) OnUpdateMiddlewareHandler

type OnDeleteMiddlewareHandler func(ctx context.Context, item *Item) (*Item, error)
type OnDeleteMiddleware func(next OnDeleteMiddlewareHandler) OnDeleteMiddlewareHandler

type OnClearMiddlewareHandler func(ctx context.Context, q *query.Query) (int, error)
type OnClearMiddleware func(next OnClearMiddlewareHandler) OnClearMiddlewareHandler

type middlewareHandlers struct {
	onGetC    []OnGetMiddleware
	onFindC   []OnFindMiddleware
	onReduceC []OnReduceMiddleware
	onInsertC []OnInsertMiddleware
	onUpdateC []OnUpdateMiddleware
	onDeleteC []OnDeleteMiddleware
	onClearC  []OnClearMiddleware

	onGetThen      OnGetMiddlewareHandler
	onMultiGetThen OnMultiGetMiddlewareHandler
	onFindThen     OnFindMiddlewareHandler
	onReduceThen   OnReduceMiddlewareHandler
	onInsertThen   OnInsertMiddlewareHandler
	onUpdateThen   OnUpdateMiddlewareHandler
	onDeleteThen   OnDeleteMiddlewareHandler
	onClearThen    OnClearMiddlewareHandler
}

func (r *Resource) Chain(middlewares ...interface{}) {
	for _, m := range middlewares {
		switch m := m.(type) {
		case []interface{}:
			r.Chain(m...)

		case OnGetMiddleware:
			r.middlewares.onGetC = append(r.middlewares.onGetC, m)
			r.middlewares.onGetThen = onGetMiddlewareDefault(r)
			for i := len(r.middlewares.onGetC) - 1; i >= 0; i-- {
				r.middlewares.onGetThen = r.middlewares.onGetC[i](r.middlewares.onGetThen)
			}

		// case OnMultiGetMiddleware:
		// TODO: implement if feasible

		case OnFindMiddleware:
			r.middlewares.onFindC = append(r.middlewares.onFindC, m)
			r.middlewares.onFindThen = onFindMiddlewareDefault(r)
			for i := len(r.middlewares.onFindC) - 1; i >= 0; i-- {
				r.middlewares.onFindThen = r.middlewares.onFindC[i](r.middlewares.onFindThen)
			}

		case OnReduceMiddleware:
			r.middlewares.onReduceC = append(r.middlewares.onReduceC, m)
			r.middlewares.onReduceThen = onReduceMiddlewareDefault(r)
			for i := len(r.middlewares.onReduceC) - 1; i >= 0; i-- {
				r.middlewares.onReduceThen = r.middlewares.onReduceC[i](r.middlewares.onReduceThen)
			}

		case OnInsertMiddleware:
			r.middlewares.onInsertC = append(r.middlewares.onInsertC, m)
			r.middlewares.onInsertThen = onInsertMiddlewareDefault(r)
			for i := len(r.middlewares.onInsertC) - 1; i >= 0; i-- {
				r.middlewares.onInsertThen = r.middlewares.onInsertC[i](r.middlewares.onInsertThen)
			}

		case OnUpdateMiddleware:
			r.middlewares.onUpdateC = append(r.middlewares.onUpdateC, m)
			r.middlewares.onUpdateThen = onUpdateMiddlewareDefault(r)
			for i := len(r.middlewares.onUpdateC) - 1; i >= 0; i-- {
				r.middlewares.onUpdateThen = r.middlewares.onUpdateC[i](r.middlewares.onUpdateThen)
			}

		case OnDeleteMiddleware:
			r.middlewares.onDeleteC = append(r.middlewares.onDeleteC, m)
			r.middlewares.onDeleteThen = onDeleteMiddlewareDefault(r)
			for i := len(r.middlewares.onDeleteC) - 1; i >= 0; i-- {
				r.middlewares.onDeleteThen = r.middlewares.onDeleteC[i](r.middlewares.onDeleteThen)
			}

		case OnClearMiddleware:
			r.middlewares.onClearC = append(r.middlewares.onClearC, m)
			r.middlewares.onClearThen = onClearMiddlewareDefault(r)
			for i := len(r.middlewares.onClearC) - 1; i >= 0; i-- {
				r.middlewares.onClearThen = r.middlewares.onClearC[i](r.middlewares.onClearThen)
			}

		default:
			panic(fmt.Sprintf("unknown middleware type %T", m))
		}
	}
}

func initMiddlewares(r *Resource) {
	r.middlewares.onGetThen = onGetMiddlewareDefault(r)
	r.middlewares.onMultiGetThen = onMultiGetMiddlewareDefault(r)
	r.middlewares.onFindThen = onFindMiddlewareDefault(r)
	r.middlewares.onReduceThen = onReduceMiddlewareDefault(r)
	r.middlewares.onInsertThen = onInsertMiddlewareDefault(r)
	r.middlewares.onUpdateThen = onUpdateMiddlewareDefault(r)
	r.middlewares.onDeleteThen = onDeleteMiddlewareDefault(r)
	r.middlewares.onClearThen = onClearMiddlewareDefault(r)
}

func onGetMiddlewareDefault(r *Resource) OnGetMiddlewareHandler {
	return func(ctx context.Context, id interface{}) (item *Item, err error) {
		if err = r.hooks.onGet(ctx, id); err == nil {
			item, err = r.storage.Get(ctx, id)
		}
		r.hooks.onGot(ctx, &item, &err)
		return
	}
}

func onMultiGetMiddlewareDefault(r *Resource) OnMultiGetMiddlewareHandler {
	return func(ctx context.Context, ids []interface{}) (items []*Item, err error) {
		errs := make([]error, len(ids))
		for i, id := range ids {
			errs[i] = r.hooks.onGet(ctx, id)
			if err == nil && errs[i] != nil {
				// first pre-hook error is the global error.
				err = errs[i]
			}
		}
		// Perform the storage request if none of the pre-hook returned an err.
		if err == nil {
			items, err = r.storage.MultiGet(ctx, ids)
		}
		var errOverwrite error
		for i := range ids {
			var _item *Item
			if len(items) > i {
				_item = items[i]
			}
			// Give the pre-hook error for this id or global otherwise.
			_err := errs[i]
			if _err == nil {
				_err = err
			}
			r.hooks.onGot(ctx, &_item, &_err)
			if errOverwrite == nil && _err != errs[i] {
				errOverwrite = _err // apply change done on the first error.
			}
			if _err == nil && len(items) > i && _item != items[i] {
				items[i] = _item // apply changes done by hooks if any.
			}
		}
		if errOverwrite != nil {
			err = errOverwrite
		}
		if err != nil {
			items = nil
		}
		return
	}
}

func onFindMiddlewareDefault(r *Resource) OnFindMiddlewareHandler {
	return func(ctx context.Context, q *query.Query, forceTotal bool) (list *ItemList, err error) {
		if err = r.hooks.onFind(ctx, q); err == nil {
			list, err = r.storage.Find(ctx, q)
			if err == nil && list.Total == -1 && forceTotal {
				// Send a query with no window so the storage won't be tempted to
				// count within the window.
				list.Total, err = r.storage.Count(ctx, &query.Query{Predicate: q.Predicate})
			}
		}
		r.hooks.onFound(ctx, q, &list, &err)
		return
	}
}

func onReduceMiddlewareDefault(r *Resource) OnReduceMiddlewareHandler {
	return func(ctx context.Context, q *query.Query, reducer ReducerFunc) error {
		// Since Reduce is a read only operation, and needs to be fast,
		// we don't call any hooks. Any modification on the data, needs to be
		// done in the reducer function.
		return r.storage.Reduce(ctx, q, reducer)
	}
}

func onInsertMiddlewareDefault(r *Resource) OnInsertMiddlewareHandler {
	return func(ctx context.Context, items []*Item) ([]*Item, error) {
		var err error
		if err = r.hooks.onInsert(ctx, items); err == nil {
			if err = recalcEtag(items); err == nil {
				err = r.storage.Insert(ctx, items)
			}
		}
		r.hooks.onInserted(ctx, items, &err)
		return items, err
	}
}

func onUpdateMiddlewareDefault(r *Resource) OnUpdateMiddlewareHandler {
	return func(ctx context.Context, item *Item, original *Item) (*Item, error) {
		var err error
		if err = r.hooks.onUpdate(ctx, item, original); err == nil {
			if err = recalcEtag([]*Item{item}); err == nil {
				err = r.storage.Update(ctx, item, original)
			}
		}
		r.hooks.onUpdated(ctx, item, original, &err)
		return item, err
	}
}

func onDeleteMiddlewareDefault(r *Resource) OnDeleteMiddlewareHandler {
	return func(ctx context.Context, item *Item) (*Item, error) {
		var err error
		if err = r.hooks.onDelete(ctx, item); err == nil {
			err = r.storage.Delete(ctx, item)
		}
		r.hooks.onDeleted(ctx, item, &err)
		return item, err
	}
}

func onClearMiddlewareDefault(r *Resource) OnClearMiddlewareHandler {
	return func(ctx context.Context, q *query.Query) (deleted int, err error) {
		if err = r.hooks.onClear(ctx, q); err == nil {
			deleted, err = r.storage.Clear(ctx, q)
		}
		r.hooks.onCleared(ctx, q, &deleted, &err)
		return
	}
}
