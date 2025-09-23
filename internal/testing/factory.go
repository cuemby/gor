package testing

import (
	"fmt"
	"reflect"
	"time"
)

// Factory provides a factory pattern for creating test data
type Factory struct {
	definitions map[string]FactoryDefinition
	sequences   map[string]int
}

// FactoryDefinition defines how to create a model
type FactoryDefinition struct {
	Model      interface{}
	Attributes map[string]AttributeFunc
	AfterBuild []func(interface{})
	AfterCreate []func(interface{})
}

// AttributeFunc generates an attribute value
type AttributeFunc func(*Factory) interface{}

// NewFactory creates a new factory
func NewFactory() *Factory {
	return &Factory{
		definitions: make(map[string]FactoryDefinition),
		sequences:   make(map[string]int),
	}
}

// Define defines a factory for a model
func (f *Factory) Define(name string, model interface{}, attributes map[string]AttributeFunc) {
	f.definitions[name] = FactoryDefinition{
		Model:       model,
		Attributes:  attributes,
		AfterBuild:  make([]func(interface{}), 0),
		AfterCreate: make([]func(interface{}), 0),
	}
}

// Build builds a model instance without saving
func (f *Factory) Build(name string, overrides ...map[string]interface{}) (interface{}, error) {
	def, ok := f.definitions[name]
	if !ok {
		return nil, fmt.Errorf("factory %s not defined", name)
	}

	// Create new instance
	modelType := reflect.TypeOf(def.Model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	instance := reflect.New(modelType)

	// Set attributes
	for field, attrFunc := range def.Attributes {
		value := attrFunc(f)
		f.setField(instance.Elem(), field, value)
	}

	// Apply overrides
	for _, override := range overrides {
		for field, value := range override {
			f.setField(instance.Elem(), field, value)
		}
	}

	// Run after build callbacks
	for _, callback := range def.AfterBuild {
		callback(instance.Interface())
	}

	return instance.Interface(), nil
}

// Create builds and saves a model instance
func (f *Factory) Create(name string, orm interface{}, overrides ...map[string]interface{}) (interface{}, error) {
	model, err := f.Build(name, overrides...)
	if err != nil {
		return nil, err
	}

	// Save using ORM if provided
	if orm != nil {
		if creator, ok := orm.(interface{ Create(interface{}) error }); ok {
			if err := creator.Create(model); err != nil {
				return nil, err
			}
		}
	}

	// Run after create callbacks
	if def, ok := f.definitions[name]; ok {
		for _, callback := range def.AfterCreate {
			callback(model)
		}
	}

	return model, nil
}

// BuildList builds multiple model instances
func (f *Factory) BuildList(name string, count int, overrides ...map[string]interface{}) ([]interface{}, error) {
	models := make([]interface{}, count)

	for i := 0; i < count; i++ {
		model, err := f.Build(name, overrides...)
		if err != nil {
			return nil, err
		}
		models[i] = model
	}

	return models, nil
}

// CreateList creates multiple model instances
func (f *Factory) CreateList(name string, count int, orm interface{}, overrides ...map[string]interface{}) ([]interface{}, error) {
	models := make([]interface{}, count)

	for i := 0; i < count; i++ {
		model, err := f.Create(name, orm, overrides...)
		if err != nil {
			return nil, err
		}
		models[i] = model
	}

	return models, nil
}

// Sequence generates a sequential value
func (f *Factory) Sequence(name string) int {
	f.sequences[name]++
	return f.sequences[name]
}

// setField sets a field value using reflection
func (f *Factory) setField(v reflect.Value, field string, value interface{}) {
	fieldValue := v.FieldByName(field)
	if !fieldValue.IsValid() || !fieldValue.CanSet() {
		return
	}

	if value == nil {
		return
	}

	valueReflect := reflect.ValueOf(value)
	if fieldValue.Type() != valueReflect.Type() {
		// Try to convert if possible
		if valueReflect.Type().ConvertibleTo(fieldValue.Type()) {
			valueReflect = valueReflect.Convert(fieldValue.Type())
		}
	}

	fieldValue.Set(valueReflect)
}

// Common attribute generators

// SequentialID generates sequential IDs
func SequentialID(prefix string) AttributeFunc {
	return func(f *Factory) interface{} {
		return fmt.Sprintf("%s_%d", prefix, f.Sequence(prefix))
	}
}

// SequentialEmail generates sequential emails
func SequentialEmail(domain string) AttributeFunc {
	return func(f *Factory) interface{} {
		return fmt.Sprintf("user%d@%s", f.Sequence("email"), domain)
	}
}

// RandomName generates random names
func RandomName() AttributeFunc {
	names := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Henry"}
	return func(f *Factory) interface{} {
		return names[RandomInt(0, len(names)-1)]
	}
}

// CurrentTime generates current timestamp
func CurrentTime() AttributeFunc {
	return func(f *Factory) interface{} {
		return time.Now()
	}
}

// FixedValue returns a fixed value
func FixedValue(value interface{}) AttributeFunc {
	return func(f *Factory) interface{} {
		return value
	}
}

// Association creates an associated model
func Association(factoryName string, orm interface{}) AttributeFunc {
	return func(f *Factory) interface{} {
		model, _ := f.Create(factoryName, orm)
		return model
	}
}

// DefaultFactories creates common factories
func DefaultFactories() *Factory {
	factory := NewFactory()

	// User factory
	factory.Define("user", struct {
		ID        int
		Email     string
		Name      string
		Password  string
		CreatedAt time.Time
		UpdatedAt time.Time
	}{}, map[string]AttributeFunc{
		"ID":        SequentialID("user"),
		"Email":     SequentialEmail("test.com"),
		"Name":      RandomName(),
		"Password":  FixedValue("password123"),
		"CreatedAt": CurrentTime(),
		"UpdatedAt": CurrentTime(),
	})

	// Post factory
	factory.Define("post", struct {
		ID        int
		Title     string
		Body      string
		AuthorID  int
		Published bool
		CreatedAt time.Time
		UpdatedAt time.Time
	}{}, map[string]AttributeFunc{
		"ID": SequentialID("post"),
		"Title": func(f *Factory) interface{} {
			return fmt.Sprintf("Post Title %d", f.Sequence("post_title"))
		},
		"Body": func(f *Factory) interface{} {
			return fmt.Sprintf("This is the body of post %d", f.Sequence("post_body"))
		},
		"AuthorID":  FixedValue(1),
		"Published": FixedValue(true),
		"CreatedAt": CurrentTime(),
		"UpdatedAt": CurrentTime(),
	})

	// Comment factory
	factory.Define("comment", struct {
		ID        int
		PostID    int
		AuthorID  int
		Content   string
		CreatedAt time.Time
		UpdatedAt time.Time
	}{}, map[string]AttributeFunc{
		"ID":      SequentialID("comment"),
		"PostID":  FixedValue(1),
		"AuthorID": FixedValue(1),
		"Content": func(f *Factory) interface{} {
			return fmt.Sprintf("Comment %d", f.Sequence("comment_content"))
		},
		"CreatedAt": CurrentTime(),
		"UpdatedAt": CurrentTime(),
	})

	return factory
}