import {Construct} from "@aws-cdk/core";
import * as lambda from "@aws-cdk/aws-lambda";
import {Bucket} from "@aws-cdk/aws-s3";
import {CfnCluster} from "@aws-cdk/aws-memorydb";
import * as path from "path";
import {exec} from "@aws-cdk/aws-lambda-nodejs/lib/util";

export class SfAuthMigration extends Construct {

    readonly bucket: Bucket;
    readonly memoryDbCluster: CfnCluster;
    readonly lambda: lambda.Function;

    constructor(scope: Construct, id: string, props: { bucket: Bucket, memoryDbCluster: CfnCluster }) {
        super(scope, id);

        this.bucket = props.bucket;
        this.memoryDbCluster = props.memoryDbCluster;
        this.lambda = this.getLambda();

        this.bucket.grantRead(this.lambda);
    }

    private getLambda(): lambda.Function {
        const id = 'lambda-sf-auth-migration';
        const sourceFolder = path.join(__dirname, 'lambdas/sf-auth-migration');
        return new lambda.Function(this, id, {
            runtime: lambda.Runtime.GO_1_X,
            code: lambda.Code.fromAsset(sourceFolder, {
                bundling: {
                    image: lambda.Runtime.GO_1_X.bundlingImage,
                    local: {
                        tryBundle(outputDir: string) {
                            const sync = exec('docker', ['run',
                                '-v', `${sourceFolder}:/asset-input`,
                                '-v', `${outputDir}:/asset-output`,
                                `${lambda.Runtime.GO_1_X.bundlingImage.image}`,
                                'bash', '-c', 'cd /asset-input; GOOS=linux go build -o /asset-output/main']);
                            console.log(sync);
                            return true
                        },
                    },
                    command: [],
                },
            }),
            handler: 'main',
            environment: {
                BUCKET: this.bucket.bucketName,
                MEMORY_DB_URI: this.memoryDbCluster.attrClusterEndpointAddress,
                MEMORY_DB_PORT: this.memoryDbCluster.attrClusterEndpointPort.toString()
            }
        })
    }
}