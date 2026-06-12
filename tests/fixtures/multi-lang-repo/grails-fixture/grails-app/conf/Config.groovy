grails {
    profile = 'web'
    codegen {
        defaultPackage = 'example'
    }
}

environments {
    development {
        logger.level = 'DEBUG'
    }
    production {
        logger.level = 'INFO'
    }
}
