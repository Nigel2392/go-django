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
    if (rules.length === 0) {
        rules = [tsLoaderConfig, tsxLoaderConfig]
    }
    return {
        resolve: {
            extensions: ['.ts', '.tsx', 'css', '...'],
        },
        mode: 'production',
        module: {
            rules: [
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
        ...baseConfig([
            {
                test: /\.css$/i,
                use: [
                    'style-loader',
                    'css-loader',
                ]
            },
            {
                test: /\.ts$/i,
                use: [
                    "ts-loader",
                ]
            }
        ]),
    },
    {
        entry: './django/contrib/pages/assets/static_src/index.ts',
        output: {
            'path': path.resolve(__dirname, 'django/contrib/pages/assets/static/pages/admin/js/'),
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
    {
        entry: './django/contrib/editor/static_src/index.ts',
        output: {
            'path': path.resolve(__dirname, 'django/contrib/editor/static/editorjs/js/'),
            'filename': 'index.js'
        },
        ...baseConfig(),
    },
]