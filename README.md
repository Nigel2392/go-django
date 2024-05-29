postcodes
================

**A dutch postcode lookup app for Django.**

By default we use the free (but rate limited) API from [postcode.go-dev.nl](https://postcode.go-dev.nl/).

You must create an account and get an API key to use this package (in the default configuration).

This API key must be set in the [settings](#settings).

**Features**

* Lookup addresses by postcode and home number
* Customizable form fields
* Customizable API endpoint
* Autofill form fields with the address data
* Validation of form fields when the address is not found
* Validation of form fields when user input was given (regex, min, max.)

Quick start
-----------

1. Install the package via pip:

   ```bash
   pip install django-postcodes
   ```

2. Add 'postcodes' to your INSTALLED_APPS setting like this:

   ```
   INSTALLED_APPS = [
   ...,
      'postcodes',
   ]
   ```
3. Add the postcodes URL to your project's `urls.py`:

   ```python
   path('postcodes/', include('postcodes.urls', namespace='postcodes')),
   ```


## Example usage

First we must define the appropriate form.

This is an example of a form that uses the default API endpoint (custom ones can be defined in the settings):

Every attribute can be customized, except for the postcode and home number fields, which are required for the lookup.

These are hard-coded.

All other elements provided in the `bind` object will be filled in with the data from the API.

```python
class AddressForm(forms.Form):
    # These are required for the lookup
    postcode = forms.CharField(max_length=10, widget=forms.TextInput(attrs={"placeholder": "1234 AB", "class": "postcode", "pattern": "^[0-9]{4}(\s+|)[A-Z]{2}$"}))
    home_number = forms.CharField(max_length=10, widget=forms.TextInput(attrs={"placeholder": "123", "class": "home_number"}))
    
    # Custom fields
    street = forms.CharField(max_length=255, widget=forms.TextInput(attrs={"placeholder": "Main street", "class": "street"}))
    city = forms.CharField(max_length=255, widget=forms.TextInput(attrs={"placeholder": "Amsterdam", "class": "city"}))
    municipality = forms.CharField(max_length=255, widget=forms.TextInput(attrs={"placeholder": "Amsterdam", "class": "municipality"}))
    province = forms.CharField(max_length=255, widget=forms.TextInput(attrs={"placeholder": "Noord-Holland", "class": "province"}))
    build_year = forms.IntegerField(widget=forms.NumberInput(attrs={"placeholder": "1990", "class": "build_year", "pattern": "^[0-9]{4}$", "min": "1900", "max": "2022"}))
    floor_area = forms.DecimalField(widget=forms.NumberInput(attrs={"placeholder": "100", "class": "floor_area", "pattern": "^[0-9]{1,3}$"})) # "pattern": "^[0-9]{1,3}$"
    geo_x = forms.DecimalField(widget=forms.NumberInput(attrs={"placeholder": "52.123456", "class": "geo_x"}))
    geo_y = forms.DecimalField(widget=forms.NumberInput(attrs={"placeholder": "4.123456", "class": "geo_y"}))
    rd_x = forms.DecimalField(widget=forms.NumberInput(attrs={"placeholder": "123456", "class": "rd_x"})) # Rijksdriehoek
    rd_y = forms.DecimalField(widget=forms.NumberInput(attrs={"placeholder": "123456", "class": "rd_y"})) # Rijksdriehoek
```

Then we can define our template

```html
{% extends 'base.html' %}

{% block content %}
   <link rel="stylesheet" href="{% static 'postcodes/css/postcodes.css' %}">
   <script src="{% static 'postcodes/js/postcodes.js' %}" data-api-url="{% url "postcodes:api" %}"></script>

   <form method="post">
       {% csrf_token %}
       {{ form.as_p }}
       <button type="submit">Submit</button>
   </form>

   ...

   <script>
       document.addEventListener('DOMContentLoaded', function() {
           lookupPostcode({
               bind: {
                   // These are required for the lookup
                    postcode: document.querySelector('#id_postcode'),
                    home_number: document.querySelector('#id_home_number'),

                    // Custom fields returned by the API
                    straat: document.querySelector("#id_street"),
                    woonplaats: document.querySelector("#id_city"),
                    gemeente: document.querySelector("#id_municipality"),
                    provincie: document.querySelector("#id_province"),
                    bouwjaar: document.querySelector("#id_build_year"),
                    vloeroppervlakte: document.querySelector("#id_floor_area"),

                    // Or optionally as a queryselector
                    // If everything is a string - it is safe to omit the DOMContentLoaded eventListener
                    latitude: "#id_geo_x",
                    longitude: "#id_geo_y",
                    rd_x: "#id_rd_x",
                    rd_y: "#id_rd_y",
               },
               success: function(addr) {
                   console.log(addr);
               },
               error: function(error) {
                   console.log(error);
               }
           })
       });
   </script>

{% endblock %}
```

The form will now automatically fill in the address fields when a valid postcode and home number is entered.

If it is invalid or not found, the error callback will be called.

## Settings

### `ADDR_VALIDATOR_API_KEY`

The API key to use for the postcode lookup.

This will be used by the `postcodes.postcode.address_check` function.


### `ADDR_VALIDATOR_API_URL`

The API URL to use for the postcode lookup.

This will be used by the `postcodes.postcode.address_check` function.

It should contain the `{postcode}` and `{home_number}` placeholders.


### `ADDR_VALIDATOR_PARAMETER_FORMAT`

An actual function that formats the parameters for the API URL.

This may not be a path to a function, but the function itself.

Example:

```python
def default_parameter_formatter(**kwargs):
    return "&".join([f"{key}={value}" for key, value in kwargs.items()])
```


### `ADDR_VALIDATOR_ERROR_ATTRIBUTE`

The attribute in the response that contains the error message.

This will be used by the `postcodes.postcode.address_check` function.

AddressValidationError exceptions will return the appropriate error message if the attribute is found.


### `ADDR_VALIDATOR_CACHE_TIMEOUT`

The timeout for the cache in seconds.

This will be used by the `postcodes.postcode.address_check` function.

It is highly recommended to set this to a high number; by default it caches for a week.

The default endpoint is free to use, but has a rate limit.


### `ADDR_VALIDATOR_API_KEY_ATTRIBUTE`

The attribute in the response that contains the API key.

This will be used by the `postcodes.postcode.address_check` function.

It is the header that should be sent with the request.

Defaults to `X-API-Token`.


### `ADDR_VALIDATOR_REQUIRES_AUTH`

Whether the API requires authentication.

This will be used by the internal view to check if the user is authenticated.

If the user is authenticated; the view will return the address data.

This does not matter if you use the `postcodes.postcode.address_check` function.
