package example

class Foo {
    String name
    Integer count

    static constraints = {
        name nullable: false
        count min: 0
    }
}
