package example

class FooTagLib {
    static namespace = "foo"

    def greet = { attrs ->
        out << "Hello, ${attrs.name}!"
    }
}
