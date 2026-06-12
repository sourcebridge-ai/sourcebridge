package example

class FooService {
    def findAll() {
        Foo.list()
    }

    def count() {
        Foo.count()
    }
}
