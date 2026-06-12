package example

import spock.lang.Specification

class FooSpec extends Specification {
    def "should count items"() {
        given:
        def service = new FooService()

        expect:
        service.count() >= 0
    }
}
