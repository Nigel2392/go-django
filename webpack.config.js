const path = require('path');

const tsLoaderConfig = {
    test: /\.ts$/i,
    use: 'ts-loader',
    exclude: '/node_modules/'
}

function baseConfig(rules = []) {
    return {
        resolve: {
            extensions: ['.ts', '...'],
        },
        mode: 'production',
        module: {
            rules: [
                tsLoaderConfig,
                ...rules
            ]
        }
    }
}

module.exports = [
    {
        entry: './django/contrib/blocks/assets/static_src/index.ts',
        output: {
            'path': path.resolve(__dirname, 'django/contrib/blocks/assets/static/blocks/js/'),
            'filename': 'index.js'
        },
        ...baseConfig(),
    },
]