const path = require('path');

const tsLoaderConfig = {
    test: /\.ts$/i,
    use: 'ts-loader',
    exclude: '/node_modules/'
}

const tsxLoaderConfig = {
    test: /\.tsx$/i,
    use: 'ts-loader',
    exclude: '/node_modules/'
}

function baseConfig(rules = []) {
    return {
        resolve: {
            extensions: ['.ts', '.tsx', '...'],
        },
        mode: 'production',
        module: {
            rules: [
                tsLoaderConfig,
                tsxLoaderConfig,
                ...rules
            ]
        }
    }
}

module.exports = [
    {
        entry: './django/contrib/admin/assets/static_src/index.ts',
        output: {
            'path': path.resolve(__dirname, 'django/contrib/admin/assets/static/admin/js/'),
            'filename': 'index.js'
        },
        ...baseConfig(),
    },
    {
        entry: './django/contrib/blocks/assets/static_src/index.ts',
        output: {
            'path': path.resolve(__dirname, 'django/contrib/blocks/assets/static/blocks/js/'),
            'filename': 'index.js'
        },
        ...baseConfig(),
    },
]