# go-css
**Important! This was originally forked from napsy.**<br>
*[Original Repo](https://github.com/napsy/go-css)*.

---

This parser understands simple CSS and comes with a basic CSS syntax checker. 
Does not currently support the following features:
- ~~Media queries~~  *partially supported*
- ~~Comma seperated selectors~~ *partially supported*

## Installation

```
go get github.com/itskass/go-css
```

## Example usage:

```go

import "github.com/itskass/go-css"

ex1 := `rule {
	style1: value1;
	style2: value2;
}`

stylesheet, err := css.Unmarshal([]byte(ex1))
if err != nil {
	panic(err)
}

fmt.Printf("Defined rules:\n")

for k, _ := range stylesheet {
	fmt.Printf("- rule %q\n", k)
}
```

You can get a CSS verifiable property by calling ``CSSStyle``:

```go
style, err := css.CSSStyle("background-color", styleSheet["body"])
if err != nil {
	fmt.Printf("Error checking body background color: %v\n", err)
} else {
	fmt.Printf("Body background color is %v", style)
}
```

Most of the CSS properties are currently not implemented, but you can always write your own handler by writing a ``StyleHandler`` function and adding it to the ``StylesTable`` map.

---
# (Forked)

This was forked from napsy because I required extra functionality like the ability to return all tokens amoung others.