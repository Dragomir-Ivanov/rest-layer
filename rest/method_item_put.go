package rest

import (
	"context"
	"net/http"

	clone "github.com/huandu/go-clone/generic"
	"github.com/rs/rest-layer/resource"
	"github.com/rs/rest-layer/schema"
	"github.com/rs/rest-layer/schema/query"
)

// itemPut handles PUT resquests on an item URL.
//
// Reference: http://tools.ietf.org/html/rfc2616#section-9.6
func itemPut(ctx context.Context, r *http.Request, route *RouteMatch) (status int, headers http.Header, body interface{}) {
	if route.Command() != nil {
		return itemPutCommand(ctx, r, route)
	}
	return itemPutCreateReplace(ctx, r, route)
}

func itemPutCreateReplace(ctx context.Context, r *http.Request, route *RouteMatch) (status int, headers http.Header, body interface{}) {
	var payload map[string]interface{}
	if e := decodePayload(r, &payload); e != nil {
		return e.Code, nil, e
	}
	q, e := route.Query()
	if e != nil {
		return e.Code, nil, e
	}
	rsrc := route.Resource()
	// Fetch original item if exist (PUT can be used to create a document with a
	// manual id).
	var original *resource.Item
	q.Window = &query.Window{Limit: 1}
	if l, err := rsrc.Find(ctx, q); err != nil && err != ErrNotFound {
		e, code := NewError(err)
		return code, nil, e
	} else if len(l.Items) == 1 {
		original = l.Items[0]
	}
	// Check if method is allowed based on the type of PUT:
	// - PUT on non existing item = create
	// - PUT on existing item = replace
	mode := resource.Create
	if original != nil {
		// If original is found, the mode is replace rather than create.
		mode = resource.Replace
	}
	if !rsrc.Conf().IsModeAllowed(mode) {
		status := http.StatusMethodNotAllowed
		return status, nil, &Error{status, http.StatusText(status), nil}
	}
	// If-Match / If-Unmodified-Since handling.
	if err := checkIntegrityRequest(r, original); err != nil {
		return err.Code, nil, err
	}
	status = 200
	var changes map[string]interface{}
	var base map[string]interface{}
	if original == nil {
		// PUT used to create a new document.
		changes, base = rsrc.Validator().Prepare(ctx, payload, nil, false)
		status = 201
	} else {
		// PUT used to replace an existing document.
		changes, base = rsrc.Validator().Prepare(ctx, payload, &original.Payload, true)
	}
	// Append lookup fields to base payload so it isn't caught by ReadOnly
	// (i.e.: contains id and parent resource refs if any).
	for k, v := range route.ResourcePath.Values() {
		base[k] = v
		// Also, ensure there's no tombstone set on the field
		if changes[k] == schema.Tombstone {
			delete(changes, k)
		}
	}
	doc, errs := rsrc.Validator().Validate(changes, base)
	if len(errs) > 0 {
		return 422, nil, &Error{422, "Document contains error(s)", errs}
	}
	if original != nil {
		if id, found := doc["id"]; found && id != original.ID {
			return 422, nil, &Error{422, "Cannot change document ID", nil}
		}
	}
	item, err := resource.NewItem(doc)
	if err != nil {
		e, code := NewError(err)
		return code, nil, e
	}

	preHookEtag := item.ETag
	if len(q.Projection) > 0 {
		projected, err := q.Projection.Eval(ctx, item.Payload, restResource{rsrc})
		if err != nil {
			e, code := NewError(err)
			return code, nil, e
		}
		preHookEtag, err = resource.GenEtag(projected)
		if err != nil {
			e, code := NewError(err)
			return code, nil, e
		}
	}
	// If we have an original item, pass it to the handler so we make sure
	// we are still replacing the same version of the object as handler is
	// supposed check the original etag before storing when an original object
	// is provided.
	if original != nil {
		if err = rsrc.Update(ctx, item, original); err != nil {
			e, code := NewError(err)
			return code, nil, e
		}
	} else {
		if err = rsrc.Insert(ctx, []*resource.Item{item}); err != nil {
			e, code := NewError(err)
			return code, nil, e
		}
	}

	postHookEtag := item.ETag
	// Evaluate projection so response gets the same format as read requests.
	item.Payload, err = q.Projection.Eval(ctx, item.Payload, restResource{rsrc})
	if err != nil {
		e, code := NewError(err)
		return code, nil, e
	}

	if len(q.Projection) > 0 {
		postHookEtag, err = resource.GenEtag(item.Payload)
		if err != nil {
			e, code := NewError(err)
			return code, nil, e
		}
	}

	if isNoContent(r) && preHookEtag == postHookEtag {
		item.Payload = nil
		if status == 200 {
			status = 204
		}
		// 201 will be returned as is, but with empty body
	}

	return status, nil, item
}

func itemPutCommand(ctx context.Context, r *http.Request, route *RouteMatch) (status int, headers http.Header, body interface{}) {
	var payload map[string]interface{}
	if e := decodePayload(r, &payload); e != nil {
		return e.Code, nil, e
	}

	q, e := route.Query()
	if e != nil {
		return e.Code, nil, e
	}

	// Get original item if any.
	rsrc := route.Resource()
	var original *resource.Item
	q.Window = &query.Window{Limit: 1}
	if l, err := rsrc.Find(ctx, q); err != nil {
		// If item can't be fetch, return an error.
		e, code := NewError(err)
		return code, nil, e
	} else if len(l.Items) == 0 {
		return ErrNotFound.Code, nil, ErrNotFound
	} else {
		original = l.Items[0]
	}

	// If-Match / If-Unmodified-Since handling.
	if err := checkIntegrityRequest(r, original); err != nil {
		return err.Code, nil, err
	}

	command := route.Command()
	commandHeaders, item, responseBody, err := command(ctx, r, clone.Clone(original), payload)
	if err != nil {
		e, code := NewError(err)
		return code, nil, e
	}

	changes, base := rsrc.Validator().Prepare(ctx, item.Payload, &original.Payload, true)
	if len(changes) > 0 {
		// Append lookup fields to base payload so it isn't caught by ReadOnly
		// (i.e.: contains id and parent resource refs if any).
		for k, v := range route.ResourcePath.Values() {
			base[k] = v
		}
		doc, errs := rsrc.Validator().Validate(changes, base)
		if len(errs) > 0 {
			return 422, nil, &Error{422, "Document contains error(s)", errs}
		}
		if id, found := doc["id"]; found && id != original.ID {
			return 422, nil, &Error{422, "Cannot change document ID", nil}
		}
		item, err = resource.NewItem(doc)
		if err != nil {
			e, code := NewError(err)
			return code, nil, e
		}

		// Store the modified document by providing the original doc to instruct
		// handler to ensure the stored document didn't change between in the
		// interval. An ErrPreconditionFailed will be thrown in case of race
		// condition (i.e.: another thread modified the document between the Find()
		// and the Store()).
		if err = rsrc.Update(ctx, item, original); err != nil {
			e, code := NewError(err)
			return code, nil, e
		}

		item.Payload, err = q.Projection.Eval(ctx, item.Payload, restResource{rsrc})
		if err != nil {
			e, code := NewError(err)
			return code, nil, e
		}
	}

	status = 200
	response := map[string]interface{}{
		"response": responseBody,
	}
	if !isNoContent(r) || len(changes) > 0 {
		response["item"] = item.Payload
	}
	dummyItem := item
	dummyItem.Payload = response

	return status, commandHeaders, dummyItem
}
