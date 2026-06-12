package example

class FooController {
    def index() {
        render view: 'index'
    }

    def show(Long id) {
        [foo: Foo.get(id)]
    }
}
