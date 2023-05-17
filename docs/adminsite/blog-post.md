# An example implementation of how you would implement a blog-post model in the adminsite:

The blogpost model will have a few special fields to really show you the power of our forms, and we will assume gorm as the default ORM.

The following example is with the `go-django/auth` package used for authentication.

```go
// Declare your models here.
type Post struct {
	ID      int64                `gorm:"primaryKey;autoIncrement"`
	Title   string               `gorm:"not null;size:45"`
	Content fields.MarkdownField `gorm:"not null"`

	AuthorID     int64                         `gorm:"not null"`
	AuthorSelect fields.SliceField[*auth.User] `gorm:"-" form:"label:Author,required:true:"`
	Image        fields.FileField              `gorm:"embedded;embeddedPrefix:image_"`
	Likes        PostLikes                     `gorm:"type:json" form:"-" admin:"hidden"`
}

```

We will now be showing you the methods for this post.

Because we are using a fiew special fields, like a generic `fields.SliceField` we will need to add some special methods, along with some extra logic to actually save these more powerfull fields. (This can be seen in the Save function.)

```go
// Make sure it conforms to the interface for the admin-site
// This is does not mean it conforms to the site 100%!
var _ admin.ModelInterface[*Post] = (*Post)(nil)


// Customize how the field is displayed in the admin-site list.
func (p Post) GetAuthorIDDisplay() string {
	var user, err = auth.Auth.Queries.GetUserByID(context.Background(), p.AuthorID)
	if err != nil {
		return err.Error()
	}
	return user.LoginField()
}

// Customize how the field is displayed in the admin-site list.
func (p Post) GetLikesDisplay() string {
	return strconv.Itoa(len(p.Likes)) + " like(s)"
}

// The options to select an author in the admin-site dropdown select field.
func (p *Post) GetAuthorSelectOptions() []interfaces.Option {
	var users, err = auth.Auth.Queries.GetAllUsers(context.Background())
	if err != nil {
		return nil
	}

	var options = make([]interfaces.Option, 0, users.Len()+1)
	options = append(options, fields.NewEmptyOption())
	for head := users.Head(); head != nil; head = head.Next() {
		var user = head.Value()
		user.SelectedAsOption = user.ID == p.AuthorID
		options = append(options, &user)
	}

	return options
}

// Returns the ID of this post as a string.
func (p *Post) StringID() string {
	return strconv.Itoa(int(p.ID))
}

// Fetches /ANY/ post by an ID. 
// Even though this is a method, it is only here to fetch a new instance of this struct.
func (p *Post) GetFromStringID(idStr string) (item *Post, err error) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, err
	}

	var post Post
	err = settings.Database().First(&post, id).Error
	if err != nil {
		return nil, err
	}
	return &post, nil
}

// Conforms to the Saver interface to save models inside of the admin-site.
// See SaveStrict for a implementation.
func (p *Post) Save(creating bool) error {
	return p.SaveStrict(creating)
}

var filer = fs.NewFiler("./assets/media")

func (p *Post) SaveStrict(create bool, fields ...string) error {
	var selectedUser = p.AuthorSelect.Selected()

	// Sets the author if one was selected.
	// If none was selected; returns an error.
	if selectedUser != nil && selectedUser.ID != 0 {
		p.AuthorID = selectedUser.ID
	} else if selectedUser == nil {
		return errors.New("no author selected")
	}

	// Save the image to the filesystem, and the URL and path of the image to the database later on.
	if p.Image.File != nil {
		// Save the post under assets/media/<<POST_ID>>/<<FILENAME>>
		// The second argument is the media URL for the post.
		// Example: 127.0.0.1/media
		var err = p.Image.Save(filer, "/media", fmt.Sprintf("post_%d", p.ID))
		if err != nil {
			return err
		}
	}

	// Save the post to the database
	if create || p.ID == 0 {
		return settings.Database().Create(p).Error
	}
	if len(fields) < 1 {
		return settings.Database().Model(p).Where(p.ID).Updates(p).Error
	}
	return settings.Database().Model(p).Where(p.ID).Select(fields).Updates(p).Error
}

```

Following that; we will register the model in the adminsite.

```go
admin.Register(admin.AdminOptions[*blog.Post]{
	FormFields: []string{"Title", "Content", "Image", "AuthorSelect"},
	ListFields: []string{"Title", "Content", "Image", "AuthorID", "Likes"},
	Model:      &blog.Post{},
})
```

Now; some of these fields conform to the `views/interfaces/interfaces.go@Scripter.Script()` interface.

This means that these fields will provide some javascript code; or script tags for the fields to be rendered.

We have implemented a pre-made route for the  default fields implemented in Go-Django, but this route still needs to be registered inside a router.

```go
App.Router.AddGroup(router.NewFSRoute("/media/", "media", os.DirFS(settings.MEDIA_ROOT))) // To be able to view the media files.
App.Router.AddGroup(fields.StaticHandler) // To be able to fetch the scripts provided in script tags for `views/fields`.
```
